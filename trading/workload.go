package trading

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"
)

const (
	workloadControllerLoopTick = 1 * time.Minute
	candleTickerIdleTimeout    = 10 * time.Second
	workloadActionLoopTick     = 5 * time.Second
	entryOrderValidityTime     = 1 * time.Minute
	signalGeneratorPauseTime   = 5 * time.Minute
)

type Workload struct {
	ID      ID
	Account *Account
	Pair    Pair
}

type WorkloadRepository interface {
	CreateWorkload(workload *Workload) error

	Workloads() ([]*Workload, error)
}

type WorkloadController struct {
	workloadRepository WorkloadRepository
	idService          IDService
	exchangeConnector  ExchangeConnector
	candleRepository   CandleRepository
	signalGenerator    SignalGenerator
	positionRepository PositionRepository
	orderRepository    OrderRepository

	workloadsMutex sync.Mutex
	workloads      map[string]*WorkloadRunner

	logger Logger
}

func RunWorkloadController(
	ctx context.Context,
	workloadRepository WorkloadRepository,
	idService IDService,
	exchangeConnector ExchangeConnector,
	candleRepository CandleRepository,
	signalGenerator SignalGenerator,
	positionRepository PositionRepository,
	orderRepository OrderRepository,
	logger Logger,
) *WorkloadController {
	workerController := &WorkloadController{
		workloadRepository: workloadRepository,
		idService:          idService,
		exchangeConnector:  exchangeConnector,
		candleRepository:   candleRepository,
		signalGenerator:    signalGenerator,
		positionRepository: positionRepository,
		orderRepository:    orderRepository,
		workloads:          make(map[string]*WorkloadRunner),
	}

	go workerController.loop(ctx)

	return workerController
}

// TODO: Add a possibility to disable the workload.
func (wc *WorkloadController) loop(ctx context.Context) {
	ticker := time.NewTicker(workloadControllerLoopTick)

	for {
		select {
		case <-ticker.C:
			workloads, err := wc.workloadRepository.Workloads()
			if err != nil {
				wc.logger.Errorf("could not get workloads: [%v]", err)
				continue
			}

			wc.workloadsMutex.Lock()

			for _, workload := range workloads {
				if _, running := wc.workloads[workload.ID.String()]; running {
					continue
				}

				workloadLogger := wc.logger.WithField(
					"workloadID",
					workload.ID.String(),
				)

				exchangeService, err := wc.exchangeConnector.Connect(
					ctx,
					workload,
				)
				if err != nil {
					workloadLogger.Errorf(
						"could not connect exchange service: [%v]",
						err,
					)
					continue
				}

				workloadRunner := RunWorkload(
					ctx,
					workload,
					wc.idService,
					exchangeService,
					wc.candleRepository,
					wc.signalGenerator,
					wc.positionRepository,
					wc.orderRepository,
					workloadLogger,
				)

				wc.workloads[workload.ID.String()] = workloadRunner

				go func() {
					select {
					case err := <-workloadRunner.ErrChan():
						workloadLogger.Errorf(
							"workload terminated with error: [%v]",
							err,
						)
					case <-ctx.Done():
					}

					wc.workloadsMutex.Lock()
					delete(wc.workloads, workload.ID.String())
					wc.workloadsMutex.Unlock()
				}()
			}

			wc.workloadsMutex.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

type WorkloadRunner struct {
	workload *Workload

	idService          IDService
	exchangeService    ExchangeService
	candleRepository   CandleRepository
	signalGenerator    SignalGenerator
	positionRepository PositionRepository
	orderRepository    OrderRepository

	logger  Logger
	errChan chan error

	lastSignalTime time.Time
}

func RunWorkload(
	ctx context.Context,
	workload *Workload,
	idService IDService,
	exchangeService ExchangeService,
	candleRepository CandleRepository,
	signalGenerator SignalGenerator,
	positionRepository PositionRepository,
	orderRepository OrderRepository,
	logger Logger,
) *WorkloadRunner {
	workloadRunner := &WorkloadRunner{
		workload:           workload,
		idService:          idService,
		exchangeService:    exchangeService,
		candleRepository:   candleRepository,
		signalGenerator:    signalGenerator,
		positionRepository: positionRepository,
		orderRepository:    orderRepository,
		logger:             logger,
		errChan:            make(chan error, 1),
		lastSignalTime:     time.Now(),
	}

	loopCtx, cancelLoopCtx := context.WithCancel(ctx)

	go func() {
		workloadRunner.dataLoop(loopCtx)
		cancelLoopCtx()
	}()

	go func() {
		workloadRunner.actionLoop(loopCtx)
		cancelLoopCtx()
	}()

	return workloadRunner
}

func (wr *WorkloadRunner) dataLoop(ctx context.Context) {
	defer wr.candleRepository.DeleteCandles(wr.workload.ID.String())

	end := time.Now()
	// Must be set with respect to the CandleInterval constant.
	start := end.Add(-1 * CandleWindowSize * time.Minute)

	candles, err := wr.exchangeService.Candles(ctx, start, end)
	if err != nil {
		wr.errChan <- fmt.Errorf("failed to get candles: [%v]", err)
		return
	}

	wr.logger.Debugf("fetched [%v] historical candles", len(candles))

	wr.candleRepository.SaveCandles(wr.workload.ID.String(), candles...)

	tickerIdleTimer := time.NewTimer(candleTickerIdleTimeout)
	tickerChan, tickerErrChan := wr.exchangeService.CandlesTicker(ctx)

	for {
		select {
		case tick := <-tickerChan:
			wr.logger.Debugf("received candle tick [%v]", tick)

			wr.candleRepository.SaveCandles(
				wr.workload.ID.String(),
				tick.Candle,
			)

			if !tickerIdleTimer.Stop() {
				<-tickerIdleTimer.C
			}
			tickerIdleTimer.Reset(candleTickerIdleTimeout)
		case <-tickerIdleTimer.C:
			wr.errChan <- fmt.Errorf("ticker idle timeout expired")
			return
		case err := <-tickerErrChan:
			wr.errChan <- fmt.Errorf("ticker error: [%v]", err)
			return
		case <-ctx.Done():
			return
		}
	}
}

func (wr *WorkloadRunner) actionLoop(ctx context.Context) {
	ticker := time.NewTicker(workloadActionLoopTick)

	for {
		select {
		case <-ticker.C:
			signalGeneratorPaused := time.Now().Before(
				wr.lastSignalTime.Add(signalGeneratorPauseTime),
			)

			if !signalGeneratorPaused {
				candles := wr.candleRepository.Candles(wr.workload.ID.String())

				if signal, exists := wr.signalGenerator.Evaluate(
					candles,
				); exists {
					wr.lastSignalTime = time.Now()

					if err := wr.processSignal(ctx, signal); err != nil {
						wr.errChan <- fmt.Errorf(
							"error while processing new signal: [%v]",
							err,
						)
						return
					}
				}
			}

			orders, err := wr.refreshOrdersQueue()
			if err != nil {
				wr.errChan <- fmt.Errorf(
					"error while refreshing orders queue: [%v]",
					err,
				)
				return
			}

			for _, order := range orders {
				alreadyExecuted, err := wr.exchangeService.IsOrderExecuted(
					ctx,
					order,
				)
				if err != nil {
					wr.errChan <- fmt.Errorf(
						"error while checking order execution: [%v]",
						err,
					)
					return
				}

				if alreadyExecuted {
					if err := wr.recordOrderExecution(order); err != nil {
						wr.errChan <- fmt.Errorf(
							"error while recording order execution: [%v]",
							err,
						)
						return
					}
					continue
				}

				executed, err := wr.exchangeService.ExecuteOrder(ctx, order)
				if err != nil {
					wr.errChan <- fmt.Errorf(
						"error while executing order: [%v]",
						err,
					)
					return
				}

				if executed {
					if err := wr.recordOrderExecution(order); err != nil {
						wr.errChan <- fmt.Errorf(
							"error while recording order execution: [%v]",
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

func (wr *WorkloadRunner) processSignal(
	ctx context.Context,
	signal *Signal,
) error {
	wr.logger.Infof("received signal [%v]", signal)

	balances, err := wr.exchangeService.AccountBalances(ctx)
	if err != nil {
		return fmt.Errorf("could not get account balances: [%v]", err)
	}

	takerCommission, err := wr.exchangeService.AccountTakerCommission(ctx)
	if err != nil {
		return fmt.Errorf("could not get account commission: [%v]", err)
	}

	walletItem := &AccountWalletItem{
		Account:         wr.workload.Account,
		Asset:           wr.workload.Pair.Quote,
		Balance:         balances.BalanceOf(wr.workload.Pair.Quote),
		TakerCommission: takerCommission,
	}

	positionOpener := &PositionOpener{
		workloadID:         wr.workload.ID,
		walletItem:         walletItem,
		positionRepository: wr.positionRepository,
		idService:          wr.idService,
	}
	position, dropped, err := positionOpener.OpenPosition(signal)
	if err != nil {
		return fmt.Errorf("could not open position: [%v]", err)
	}

	if len(dropped) > 0 {
		wr.logger.Warningf("dropping signal because: [%v]", dropped)
		return nil
	}

	orderFactory := &OrderFactory{
		orderRepository: wr.orderRepository,
		idService:       wr.idService,
	}
	_, err = orderFactory.CreateEntryOrder(position)
	if err != nil {
		return fmt.Errorf(
			"could not create entry order for position [%v]: [%v]",
			position.ID,
			err,
		)
	}

	wr.logger.Infof(
		"position [%v] based on signal [%v] "+
			"has been opened successfully",
		position.ID,
		signal,
	)

	return nil
}

func (wr *WorkloadRunner) refreshOrdersQueue() ([]*Order, error) {
	openPositions, err := wr.positionRepository.Positions(
		PositionFilter{
			WorkloadID: wr.workload.ID,
			Status:     StatusOpen,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not get open positions: [%v]", err)
	}

	sort.SliceStable(openPositions, func(i, j int) bool {
		return openPositions[i].Time.Before(openPositions[j].Time)
	})

	currentPrice, err := wr.lastClosePrice()
	if err != nil {
		return nil, fmt.Errorf(
			"could not determine current price: [%v]",
			err,
		)
	}

	positionCloser := &PositionCloser{wr.positionRepository}
	orderFactory := &OrderFactory{
		orderRepository: wr.orderRepository,
		idService:       wr.idService,
	}

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
			if time.Now().Sub(entryOrder.Time) > entryOrderValidityTime {
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

func (wr *WorkloadRunner) recordOrderExecution(order *Order) error {
	wr.logger.Infof(
		"recording order [%v] execution",
		order.ID,
	)

	recorder := &OrderExecutionRecorder{wr.orderRepository}
	if err := recorder.recordOrderExecution(order); err != nil {
		return fmt.Errorf(
			"could not record order [%v] execution: [%v]",
			order.ID,
			err,
		)
	}

	return nil
}

func (wr *WorkloadRunner) lastClosePrice() (*big.Float, error) {
	candles := wr.candleRepository.Candles(wr.workload.ID.String())

	price := new(big.Float)
	err := price.UnmarshalText([]byte(candles[len(candles)-1].ClosePrice))
	if err != nil {
		return nil, err
	}

	return price, nil
}

func (wr *WorkloadRunner) ErrChan() <-chan error {
	return wr.errChan
}
