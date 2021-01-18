package core

import (
	technical "github.com/sdcoffey/techan"
)

type ordersRegistry struct {
	delegate *technical.TradingRecord
}

func newOrdersRegistry() *ordersRegistry {
	return &ordersRegistry{technical.NewTradingRecord()}
}
