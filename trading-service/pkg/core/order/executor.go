package order

import (
	"context"
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/logger"
	"math/big"
	"time"
)

const executorTick = 1 * time.Second

type Side int

const (
	BUY Side = iota
	SELL
)

func (s Side) String() string {
	switch s {
	case BUY:
		return "BUY"
	case SELL:
		return "SELL"
	}

	return ""
}

type Order struct {
	Side          Side
	Price         *big.Float
	Amount        *big.Float
	ExecutionTime time.Time
}

func (o *Order) String() string {
	return fmt.Sprintf(
		"side: %v, amount: %v, price: %v",
		o.Side,
		o.Amount.String(),
		o.Price.String(),
	)
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

func RunExecutor(
	ctx context.Context,
	logger logger.Logger,
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

				logger.Infof(
					"executing new order [%v]",
					order,
				)

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

				logger.Infof(
					"order [%v] has been executed successfully",
					order,
				)
			case <-ctx.Done():
				return
			}
		}
	}()

	return &Executor{
		ErrorChannel: errorChannel,
	}
}
