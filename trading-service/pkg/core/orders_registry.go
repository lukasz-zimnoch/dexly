package core

import (
	"github.com/sdcoffey/big"
	technical "github.com/sdcoffey/techan"
	"sync"
)

type ordersRegistry struct {
	mutex    sync.RWMutex
	delegate *technical.TradingRecord
}

func newOrdersRegistry() *ordersRegistry {
	return &ordersRegistry{
		delegate: technical.NewTradingRecord(),
	}
}

func (or *ordersRegistry) add(order *Order) {
	or.mutex.Lock()
	defer or.mutex.Unlock()

	// TODO: save to db
	// TODO: mutex on read

	or.delegate.Operate(convertToTechnicalOrder(order))
}

func convertToTechnicalOrder(order *Order) technical.Order {
	return technical.Order{
		Side:          technical.OrderSide(order.Side),
		Security:      "",
		Price:         big.NewFromString(order.Price.String()),
		Amount:        big.NewFromString(order.Amount.String()),
		ExecutionTime: order.ExecutionTime,
	}
}
