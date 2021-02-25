package trade

import (
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
	// TODO: fields
}

func NewManager(
	priceSupplier priceSupplier,
	accountSupplier accountSupplier,
	repository Repository,
) *Manager {
	// TODO: set fields
	return nil
}

func (m *Manager) NotifySignal(signal *Signal) {
	// TODO: implementation
	panic("implement me")
}

func (m *Manager) NotifyExecution(order *Order) {
	// TODO: implementation
	panic("implement me")
}

func (m *Manager) OrderQueue() []*Order {
	// TODO: implementation
	panic("implement me")
}
