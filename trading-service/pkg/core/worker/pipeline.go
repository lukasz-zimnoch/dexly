package worker

import (
	"context"
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/logger"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/trade"
	"time"
)

const pipelineTick = 5 * time.Second

type signaler interface {
	Evaluate() (*trade.Signal, bool)
}

type executor interface {
	ExecuteOrder(order *trade.Order) error

	IsOrderExecuted(order *trade.Order) (bool, error)
}

type pipeline struct {
	logger   logger.Logger
	signaler signaler
	manager  *trade.Manager
	executor executor
	errChan  chan error
}

func runPipeline(
	ctx context.Context,
	logger logger.Logger,
	signaler signaler,
	manager *trade.Manager,
	executor executor,
) *pipeline {
	pipeline := &pipeline{
		logger:   logger,
		signaler: signaler,
		manager:  manager,
		executor: executor,
		errChan:  make(chan error, 1),
	}

	go pipeline.loop(ctx)

	return pipeline
}

func (p *pipeline) loop(ctx context.Context) {
	ticker := time.NewTicker(pipelineTick)

	for {
		select {
		case <-ticker.C:
			if signal, exists := p.signaler.Evaluate(); exists {
				p.manager.NotifySignal(signal)
			}

			for _, order := range p.manager.OrderQueue() {
				alreadyExecuted, err := p.executor.IsOrderExecuted(order)
				if err != nil {
					p.errChan <- fmt.Errorf(
						"could not check order execution: [%v]",
						err,
					)
					return
				}

				if !alreadyExecuted {
					err := p.executor.ExecuteOrder(order)
					if err != nil {
						p.errChan <- fmt.Errorf(
							"could execute order: [%v]",
							err,
						)
						return
					}
				}

				p.manager.NotifyExecution(order)
			}
		case <-ctx.Done():
			return
		}
	}
}
