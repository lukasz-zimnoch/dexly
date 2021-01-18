package core

import (
	"context"
	"time"
)

const ordersManagerTick = 1 * time.Second

type candlesSource interface {
	get() []*Candle
}

type orderRequest struct {
	mock int
}

func runOrdersManager(
	ctx context.Context,
	source candlesSource,
	_ *ordersRegistry,
) <-chan *orderRequest {
	outgoingOrderRequestChannel := make(chan *orderRequest)

	go func() {
		ticker := time.NewTicker(ordersManagerTick)

		for {
			select {
			case <-ticker.C:
				_ = evaluateStrategy(source.get())
				// TODO: implementation
			case <-ctx.Done():
				return
			}
		}
	}()

	return outgoingOrderRequestChannel
}
