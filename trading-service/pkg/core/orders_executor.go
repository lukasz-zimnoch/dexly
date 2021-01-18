package core

import (
	"context"
)

func runOrdersExecutor(
	ctx context.Context,
	exchange ExchangeClient,
) chan<- *orderRequest {
	incomingOrderRequestChannel := make(chan *orderRequest)

	go func() {
		for {
			select {
			case _ = <-incomingOrderRequestChannel:
				// TODO: implementation
			case <-ctx.Done():
				return
			}
		}
	}()

	return incomingOrderRequestChannel
}
