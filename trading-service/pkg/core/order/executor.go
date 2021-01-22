package order

import (
	"context"
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/logger"
	"time"
)

const executorTick = 1 * time.Second

type Submitter interface {
	SubmitOrder(order *Order) error
}

type Executor struct {
	logger    logger.Logger
	generator *Generator
	registry  *Registry
	submitter Submitter
	errChan   chan error
}

func RunExecutor(
	ctx context.Context,
	logger logger.Logger,
	generator *Generator,
	registry *Registry,
	submitter Submitter,
) *Executor {
	executor := &Executor{
		logger:    logger,
		generator: generator,
		registry:  registry,
		submitter: submitter,
		errChan:   make(chan error, 1),
	}

	go executor.loop(ctx)

	return executor
}

func (e *Executor) loop(ctx context.Context) {
	ticker := time.NewTicker(executorTick)

	for {
		select {
		case <-ticker.C:
			order, ok := e.generator.Generate()
			if !ok {
				continue
			}

			e.logger.Infof(
				"executing new order [%v]",
				order,
			)

			err := e.submitter.SubmitOrder(order)
			if err != nil {
				// TODO: handle retries
				e.errChan <- fmt.Errorf(
					"could not submit order: [%v]",
					err,
				)
				return
			}

			e.registry.AddOrder(order)
			e.generator.RecordExecution(order)

			e.logger.Infof(
				"order [%v] has been executed successfully",
				order,
			)
		case <-ctx.Done():
			return
		}
	}
}

func (e *Executor) ErrChan() <-chan error {
	return e.errChan
}
