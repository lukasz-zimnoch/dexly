package core

import (
	"context"
)

type ordersExecutor struct {
	requestChannel chan<- *orderRequest
}

func runOrdersExecutor(
	ctx context.Context,
	exchange ExchangeClient,
	requestChannel chan *orderRequest,
) *ordersExecutor {
	go func() {
		for {
			select {
			case _ = <-requestChannel:
				// TODO: implementation
			case <-ctx.Done():
				return
			}
		}
	}()

	return &ordersExecutor{
		requestChannel: requestChannel,
	}
}
