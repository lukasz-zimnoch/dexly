package trade

import (
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/logger"
	"math/big"
	"sort"
	"time"
)

const orderValidityTime = 1 * time.Minute

type priceSupplier interface {
	Price() (*big.Float, error)
}

type accountSupplier interface {
	Balance() (*big.Float, error)

	RiskFactor() *big.Float

	TakerCommission() (*big.Float, error)
}

type PositionFilter struct {
	Pair     string
	Exchange string
	Status   PositionStatus
}

type Repository interface {
	CreatePosition(position *Position) error

	UpdatePosition(position *Position) error

	GetPositions(filter PositionFilter) ([]*Position, error)

	CountPositions(filter PositionFilter) (int, error)

	CreateOrder(order *Order) error

	UpdateOrder(order *Order) error
}

type Manager struct {
	logger             logger.Logger
	priceSupplier      priceSupplier
	accountSupplier    accountSupplier
	repository         Repository
	pair               string
	exchange           string
	openPositionsLimit int
}

func NewManager(
	logger logger.Logger,
	priceSupplier priceSupplier,
	accountSupplier accountSupplier,
	repository Repository,
	pair string,
	exchange string,
) *Manager {
	return &Manager{
		logger:             logger,
		priceSupplier:      priceSupplier,
		accountSupplier:    accountSupplier,
		repository:         repository,
		pair:               pair,
		exchange:           exchange,
		openPositionsLimit: 1,
	}
}

func (m *Manager) NotifySignal(signal *Signal) {
	m.logger.Infof("received signal [%+v]", signal)

	if signal.Type != LONG {
		m.logger.Warningf("only LONG signals are currently supported")
		return
	}

	openPositionsCount, err := m.repository.CountPositions(
		PositionFilter{
			Pair:     m.pair,
			Exchange: m.exchange,
			Status:   OPEN,
		},
	)
	if err != nil {
		m.logger.Errorf("could not count open positions: [%v]", err)
		return
	}

	if openPositionsCount >= m.openPositionsLimit {
		m.logger.Infof(
			"dropping signal [%+v] due to "+
				"open position limit restrictions",
			signal,
		)
		return
	}

	positionSize, err := m.calculatePositionSize(signal)
	if err != nil {
		m.logger.Errorf("could not calculate position size: [%v]", err)
		return
	}

	position, err := m.openPosition(signal, positionSize)
	if err != nil {
		m.logger.Errorf("could not open position: [%v]", err)
		return
	}

	_, err = m.createEntryOrder(position)
	if err != nil {
		m.logger.Errorf(
			"could not create entry order for position [%v]: [%v]",
			position.ID,
			err,
		)
		return
	}

	m.logger.Infof(
		"position [%v] based on signal [%+v] "+
			"has been opened successfully",
		position.ID,
		signal,
	)
}

func (m *Manager) calculatePositionSize(signal *Signal) (*big.Float, error) {
	accountBalance, err := m.accountSupplier.Balance()
	if err != nil {
		return nil, err
	}

	if accountBalance.Cmp(big.NewFloat(0)) == 0 {
		return nil, fmt.Errorf("account balance is zero")
	}

	accountRisk := new(big.Float).Mul(
		accountBalance,
		m.accountSupplier.RiskFactor(),
	)

	tradeRisk := new(big.Float).Sub(signal.EntryTarget, signal.StopLossTarget)

	positionSize := new(big.Float).Quo(accountRisk, tradeRisk)

	maxPositionSize := new(big.Float).Quo(accountBalance, signal.EntryTarget)
	if positionSize.Cmp(maxPositionSize) == 1 {
		positionSize = maxPositionSize
	}

	return positionSize, nil
}

func (m *Manager) openPosition(
	signal *Signal,
	positionSize *big.Float,
) (*Position, error) {
	accountCommission, err := m.accountSupplier.TakerCommission()
	if err != nil {
		return nil, err
	}

	takeProfitPrice := new(big.Float).Mul(
		signal.TakeProfitTarget,
		new(big.Float).Add(big.NewFloat(1), accountCommission),
	)

	stopLossPrice := new(big.Float).Mul(
		signal.StopLossTarget,
		new(big.Float).Sub(big.NewFloat(1), accountCommission),
	)

	position := NewPosition(
		signal.Type,
		signal.EntryTarget,
		positionSize,
		takeProfitPrice,
		stopLossPrice,
		m.pair,
		m.exchange,
	)

	err = m.repository.CreatePosition(position)
	if err != nil {
		return nil, err
	}

	return position, nil
}

func (m *Manager) closePosition(position *Position) error {
	position.Status = CLOSED

	err := m.repository.UpdatePosition(position)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) createEntryOrder(
	position *Position,
) (*Order, error) {
	order := NewOrder(
		position,
		position.Type.EntryOrderSide(),
		position.EntryPrice,
		position.Size,
	)

	err := m.repository.CreateOrder(order)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (m *Manager) createExitOrder(
	position *Position,
	price *big.Float,
) (*Order, error) {
	order := NewOrder(
		position,
		position.Type.ExitOrderSide(),
		price,
		position.Size,
	)

	err := m.repository.CreateOrder(order)
	if err != nil {
		return nil, err
	}

	return order, nil
}

// TODO: check the actually executed price and size
func (m *Manager) NotifyExecution(order *Order) {
	m.logger.Infof(
		"received notification about order [%v] execution",
		order.ID,
	)

	order.Executed = true

	if err := m.repository.UpdateOrder(order); err != nil {
		m.logger.Errorf(
			"could not update order [%v] execution state: [%v]",
			order.ID,
			err,
		)
		return
	}

	m.logger.Infof(
		"execution of order [%v] has been noted successfully",
		order.ID,
	)
}

func (m *Manager) RefreshOrdersQueue() []*Order {
	openPositions, err := m.repository.GetPositions(
		PositionFilter{
			Pair:     m.pair,
			Exchange: m.exchange,
			Status:   OPEN,
		},
	)
	if err != nil {
		m.logger.Errorf("could not get open positions: [%v]", err)
		return nil
	}

	sort.SliceStable(openPositions, func(i, j int) bool {
		return openPositions[i].Time.Before(openPositions[j].Time)
	})

	currentPrice, err := m.priceSupplier.Price()
	if err != nil {
		m.logger.Errorf("could not determine current price: [%v]", err)
		return nil
	}

	pendingOrders := make([]*Order, 0)

	for _, position := range openPositions {
		entryOrder, exitOrder, err := ordersBreakdown(position)
		if err != nil {
			m.logger.Errorf(
				"inconsistent orders state for position [%v]: [%v]",
				position.ID,
				err,
			)
			continue
		}

		if entryOrder == nil {
			// just close without trying to recover the entry order
			if err := m.closePosition(position); err != nil {
				m.logger.Errorf(
					"could not close position [%v]: [%v]",
					position.ID,
					err,
				)
			}
			continue
		}

		if !entryOrder.Executed {
			if time.Now().Sub(entryOrder.Time) > orderValidityTime {
				if err := m.closePosition(position); err != nil {
					m.logger.Errorf(
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
				exitOrder, err := m.createExitOrder(position, currentPrice)
				if err != nil {
					m.logger.Errorf(
						"could not create exit order "+
							"for position [%v]: [%v]",
						position.ID,
						err,
					)
					continue
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

		if err := m.closePosition(position); err != nil {
			m.logger.Errorf(
				"could not close position [%v]: [%v]",
				position.ID,
				err,
			)
		}
	}

	return pendingOrders
}

func ordersBreakdown(position *Position) (*Order, *Order, error) {
	ordersCount := len(position.Orders)

	if ordersCount == 0 {
		return nil, nil, nil
	} else if ordersCount == 1 {
		entryOrder := position.Orders[0]

		if entryOrder.Side != position.Type.EntryOrderSide() {
			return nil, nil, fmt.Errorf("entry order has wrong side")
		}

		return entryOrder, nil, nil
	} else if ordersCount == 2 {
		sort.SliceStable(position.Orders, func(i, j int) bool {
			return position.Orders[i].Time.Before(position.Orders[j].Time)
		})

		entryOrder := position.Orders[0]
		exitOrder := position.Orders[1]

		if entryOrder.Side != position.Type.EntryOrderSide() {
			return nil, nil, fmt.Errorf("entry order has wrong side")
		}

		if !entryOrder.Executed {
			return nil, nil, fmt.Errorf(
				"exit order exists despite entry order is not executed yet",
			)
		}

		if exitOrder.Side != position.Type.ExitOrderSide() {
			return nil, nil, fmt.Errorf("exit order has wrong side")
		}

		return entryOrder, exitOrder, nil
	} else {
		return nil, nil, fmt.Errorf("wrong orders count: [%v]", ordersCount)
	}
}
