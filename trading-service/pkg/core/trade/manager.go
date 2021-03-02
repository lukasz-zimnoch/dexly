package trade

import (
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/logger"
	"math/big"
)

type priceSupplier interface {
	Price() (*big.Float, error)
}

type accountSupplier interface {
	Balance() (*big.Float, error)

	RiskFactor() *big.Float
}

type Repository interface {
	CreatePosition(position *Position) error

	CreateOrder(order *Order) error

	UpdateOrder(order *Order) error

	GetOrders(pair string, exchange string, executed bool) ([]*Order, error)
}

type Manager struct {
	logger          logger.Logger
	priceSupplier   priceSupplier
	accountSupplier accountSupplier
	repository      Repository
	pair            string
	exchange        string
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
		logger:          logger,
		priceSupplier:   priceSupplier,
		accountSupplier: accountSupplier,
		repository:      repository,
		pair:            pair,
		exchange:        exchange,
	}
}

func (m *Manager) NotifySignal(signal *Signal) {
	// TODO: support SHORT signals as well
	if signal.Type != LONG {
		m.logger.Warningf("only LONG signals are currently supported")
		return
	}

	// TODO: check if active positions limit is not exceeded

	positionSize, err := m.calculatePositionSize(signal)
	if err != nil {
		m.logger.Errorf("could not calculate position size: [%v]", err)
		return
	}

	position := NewPosition(
		signal.Type,
		signal.EntryTarget,
		positionSize,
		signal.TakeProfitTarget,
		signal.StopLossTarget,
		m.pair,
		m.exchange,
	)

	if err = m.repository.CreatePosition(position); err != nil {
		m.logger.Errorf(
			"could not persist position [%v]: [%v]",
			position.ID,
			err,
		)
		return
	}

	order := NewOrder(position, BUY, position.EntryPrice, position.Size)

	if err = m.repository.CreateOrder(order); err != nil {
		m.logger.Errorf(
			"could not persist order [%v] for position [%v]: [%v]",
			order.ID,
			position.ID,
			err,
		)
		return
	}
}

func (m *Manager) calculatePositionSize(signal *Signal) (*big.Float, error) {
	accountBalance, err := m.accountSupplier.Balance()
	if err != nil {
		return nil, err
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

func (m *Manager) NotifyExecution(order *Order) {
	if order.Executed {
		m.logger.Warningf("order [%v] has been already executed", order.ID)
		return
	}

	order.Executed = true

	if err := m.repository.UpdateOrder(order); err != nil {
		m.logger.Errorf(
			"could not update order [%v] execution state: [%v]",
			order.ID,
			err,
		)
		return
	}
}

// TODO: consider renaming as method will have side effects
func (m *Manager) OrderQueue() []*Order {
	// TODO: take positions without any orders and create orders for them

	// TODO: get last price, compare with active positions TP and SL levels
	//  and create appropriate close orders

	orders, err := m.repository.GetOrders(m.pair, m.exchange, false)
	if err != nil {
		m.logger.Errorf("could not get pending orders: [%v]", err)
		return nil
	}

	return orders
}
