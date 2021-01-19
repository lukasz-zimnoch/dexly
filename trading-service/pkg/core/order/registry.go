package order

import (
	"sync"
)

type Registry struct {
	ordersMutex sync.RWMutex
	orders      []*Order
}

func NewRegistry() *Registry {
	return &Registry{
		orders: make([]*Order, 0),
	}
}

func (r *Registry) Add(order *Order) {
	r.ordersMutex.Lock()
	defer r.ordersMutex.Unlock()

	r.orders = append(r.orders, order)

	// TODO: save to db
}
