package order

import (
	"context"
	"fmt"
	"math/big"
	"time"
)

const executorTick = 1 * time.Second

type Side int

const (
	BUY Side = iota
	SELL
)

type Order struct {
	Side          Side
	Price         *big.Float
	Amount        *big.Float
	ExecutionTime time.Time
}

type Strategy interface {
	Propose() (*Order, bool)
	Record(order *Order)
}

type Submitter interface {
	SubmitOrder(order *Order) error
}

type Executor struct {
	ErrorChannel <-chan error
}

// TODO: add logging
func RunExecutor(
	ctx context.Context,
	strategy Strategy,
	registry *Registry,
	submitter Submitter,
) *Executor {
	errorChannel := make(chan error, 1)

	go func() {
		ticker := time.NewTicker(executorTick)

		for {
			select {
			case <-ticker.C:
				order, ok := strategy.Propose()
				if !ok {
					continue
				}

				err := submitter.SubmitOrder(order)
				if err != nil {
					// TODO: handle retries
					errorChannel <- fmt.Errorf(
						"could not submit order: [%v]",
						err,
					)
					return
				}

				registry.Add(order)
				strategy.Record(order)
			case <-ctx.Done():
				return
			}
		}
	}()

	return &Executor{
		ErrorChannel: errorChannel,
	}
}
