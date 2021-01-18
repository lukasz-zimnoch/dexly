package core

import (
	"context"
	"math/big"
	"time"
)

const ordersExecutorTick = 1 * time.Second

type orderSide int

const (
	BUY orderSide = iota
	SELL
)

type Order struct {
	Side          orderSide
	Price         *big.Float
	Amount        *big.Float
	ExecutionTime time.Time
}

type candlesSource interface {
	get() []*Candle
}

// TODO: add logging
func runOrdersExecutor(
	ctx context.Context,
	exchange ExchangeClient,
	candlesSource candlesSource,
	ordersRegistry *ordersRegistry,
) {
	go func() {
		ticker := time.NewTicker(ordersExecutorTick)

		for {
			select {
			case <-ticker.C:
				strategy := evaluateStrategy(candlesSource.get())
				signal, ok := strategy.run(ordersRegistry)
				if !ok {
					continue
				}

				order := &Order{
					Side:   signal.side,
					Price:  signal.price,
					Amount: signal.amount,
				}

				// exchange.PlaceOrder(order)

				ordersRegistry.add(order)
			case <-ctx.Done():
				return
			}
		}
	}()
}
