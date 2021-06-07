package trading

import (
	"context"
	"fmt"
	"sort"
	"time"
)

const (
	workerTick        = 5 * time.Second
	orderValidityTime = 1 * time.Minute
)

type Worker struct {
	logger             Logger
	account            *Account
	pair               Pair
	signalGenerator    SignalGenerator
	candleRepository   CandleRepository
	positionRepository PositionRepository
	orderRepository    OrderRepository
	exchange           ExchangeService
	errChan            chan error
}

func RunWorker(
	ctx context.Context,
	logger Logger,
	account *Account,
	pair Pair,
	signalGenerator SignalGenerator,
	candleRepository CandleRepository,
	positionRepository PositionRepository,
	orderRepository OrderRepository,
	exchange ExchangeService,
) *Worker {
	worker := &Worker{
		logger:             logger,
		account:            account,
		pair:               pair,
		signalGenerator:    signalGenerator,
		candleRepository:   candleRepository,
		positionRepository: positionRepository,
		orderRepository:    orderRepository,
		exchange:           exchange,
		errChan:            make(chan error, 1),
	}

	go worker.loop(ctx)

	return worker
}

func (w *Worker) loop(ctx context.Context) {
	ticker := time.NewTicker(workerTick)

	for {
		select {
		case <-ticker.C:
			if signal, exists := w.signalGenerator.Poll(); exists {
				if err := w.processSignal(ctx, signal); err != nil {
					w.errChan <- fmt.Errorf(
						"error during signal processing: [%v]",
						err,
					)
					return
				}
			}

			orders, err := w.refreshOrdersQueue()
			if err != nil {
				w.errChan <- fmt.Errorf(
					"error during orders queue refresh: [%v]",
					err,
				)
				return
			}

			for _, order := range orders {
				alreadyExecuted, err := w.exchange.IsOrderExecuted(ctx, order)
				if err != nil {
					w.errChan <- fmt.Errorf(
						"error during order execution check: [%v]",
						err,
					)
					return
				}

				if alreadyExecuted {
					if err := w.noteOrderExecution(order); err != nil {
						w.errChan <- fmt.Errorf(
							"error during noting order execution: [%v]",
							err,
						)
						return
					}
					continue
				}

				executed, err := w.exchange.ExecuteOrder(ctx, order)
				if err != nil {
					w.errChan <- fmt.Errorf(
						"error during order execution: [%v]",
						err,
					)
					return
				}

				if executed {
					if err := w.noteOrderExecution(order); err != nil {
						w.errChan <- fmt.Errorf(
							"error during noting order execution: [%v]",
							err,
						)
						return
					}
					continue
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) processSignal(ctx context.Context, signal *Signal) error {
	w.logger.Infof("received signal [%v]", signal)

	account, err := w.exchange.ExchangeAccount(ctx, w.account)
	if err != nil {
		return fmt.Errorf("could not get exchange account: [%v]", err)
	}

	positionOpener := &PositionOpener{w.positionRepository}
	position, dropped, err := positionOpener.OpenPosition(signal, account)
	if err != nil {
		return fmt.Errorf("could not open position: [%v]", err)
	}

	if len(dropped) > 0 {
		w.logger.Warningf("dropping signal because: [%v]", dropped)
		return nil
	}

	orderFactory := &OrderFactory{w.orderRepository}
	_, err = orderFactory.CreateEntryOrder(position)
	if err != nil {
		return fmt.Errorf(
			"could not create entry order for position [%v]: [%v]",
			position.ID,
			err,
		)
	}

	w.logger.Infof(
		"position [%v] based on signal [%v] "+
			"has been opened successfully",
		position.ID,
		signal,
	)

	return nil
}

func (w *Worker) refreshOrdersQueue() ([]*Order, error) {
	openPositions, err := w.positionRepository.Positions(
		PositionFilter{
			Pair:     w.pair.String(),
			Exchange: w.exchange.ExchangeName(),
			Status:   OPEN,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not get open positions: [%v]", err)
	}

	sort.SliceStable(openPositions, func(i, j int) bool {
		return openPositions[i].Time.Before(openPositions[j].Time)
	})

	currentPrice, err := w.candleRepository.LastClosePrice()
	if err != nil {
		return nil, fmt.Errorf("could not determine current price: [%v]", err)
	}

	positionCloser := &PositionCloser{w.positionRepository}
	orderFactory := &OrderFactory{w.orderRepository}

	pendingOrders := make([]*Order, 0)

	for _, position := range openPositions {
		entryOrder, exitOrder, err := position.OrdersBreakdown()
		if err != nil {
			return nil, fmt.Errorf(
				"inconsistent orders state for position [%v]: [%v]",
				position.ID,
				err,
			)
		}

		if entryOrder == nil {
			// just close without trying to recover the entry order
			if err := positionCloser.ClosePosition(position); err != nil {
				return nil, fmt.Errorf(
					"could not close position [%v]: [%v]",
					position.ID,
					err,
				)
			}
			continue
		}

		if !entryOrder.Executed {
			if time.Now().Sub(entryOrder.Time) > orderValidityTime {
				if err := positionCloser.ClosePosition(position); err != nil {
					return nil, fmt.Errorf(
						"could not close position [%v]: [%v]",
						position.ID,
						err,
					)
				}
				continue
			}

			pendingOrders = append(pendingOrders, entryOrder)
			continue
		}

		if exitOrder == nil {
			shouldExit := currentPrice.Cmp(position.StopLossPrice) <= 0 ||
				currentPrice.Cmp(position.TakeProfitPrice) >= 0

			if shouldExit {
				exitOrder, err := orderFactory.CreateExitOrder(
					position,
					currentPrice,
				)
				if err != nil {
					return nil, fmt.Errorf(
						"could not create exit order "+
							"for position [%v]: [%v]",
						position.ID,
						err,
					)
				}

				pendingOrders = append(pendingOrders, exitOrder)
				continue
			}
			continue
		}

		if !exitOrder.Executed {
			pendingOrders = append(pendingOrders, exitOrder)
			continue
		}

		if err := positionCloser.ClosePosition(position); err != nil {
			return nil, fmt.Errorf(
				"could not close position [%v]: [%v]",
				position.ID,
				err,
			)
		}
	}

	return pendingOrders, nil
}

// TODO: check the actually executed price and size
func (w *Worker) noteOrderExecution(order *Order) error {
	w.logger.Infof(
		"received note about order [%v] execution",
		order.ID,
	)

	noter := &OrderExecutionNoter{w.orderRepository}
	if err := noter.NoteOrderExecution(order); err != nil {
		return fmt.Errorf(
			"could not note order [%v] execution: [%v]",
			order.ID,
			err,
		)
	}

	w.logger.Infof(
		"execution of order [%v] has been noted successfully",
		order.ID,
	)

	return nil
}

func (w *Worker) ErrChan() <-chan error {
	return w.errChan
}
