package core

import (
	"context"
	"time"
)

const positionManagerTick = 1 * time.Second

type candlesSource interface {
	get() []*Candle
}

type positionManager struct {
	ordersRegistry *ordersRegistry
}

type orderRequest struct {
	mock int
}

func newPositionManager() *positionManager {
	return &positionManager{
		ordersRegistry: newOrdersRegistry(),
	}
}

func (pm *positionManager) run(
	ctx context.Context,
	source candlesSource,
) <-chan *orderRequest {
	orderRequestChannel := make(chan *orderRequest)

	go func() {
		ticker := time.NewTicker(positionManagerTick)

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

	return orderRequestChannel
}
