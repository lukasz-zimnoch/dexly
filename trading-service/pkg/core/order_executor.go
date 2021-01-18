package core

import (
	"context"
)

type orderExecutor struct {
	exchange ExchangeClient
}

func newOrderExecutor(exchange ExchangeClient) *orderExecutor {
	return &orderExecutor{
		exchange: exchange,
	}
}

func (oe *orderExecutor) run(ctx context.Context) chan<- *orderRequest {
	orderRequestChannel := make(chan *orderRequest)

	go func() {
		for {
			select {
			case _ = <-orderRequestChannel:
				// TODO: implementation
			case <-ctx.Done():
				return
			}
		}
	}()

	return orderRequestChannel
}
