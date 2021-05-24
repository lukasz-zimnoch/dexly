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
	ExecuteOrder(ctx context.Context, order *trade.Order) (bool, error)

	IsOrderExecuted(ctx context.Context, order *trade.Order) (bool, error)
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

			for _, order := range p.manager.RefreshOrdersQueue() {
				alreadyExecuted, err := p.executor.IsOrderExecuted(ctx, order)
				if err != nil {
					p.errChan <- fmt.Errorf(
						"could not check order execution: [%v]",
						err,
					)
					return
				}

				if alreadyExecuted {
					p.manager.NotifyExecution(order)
					continue
				}

				executed, err := p.executor.ExecuteOrder(ctx, order)
				if err != nil {
					p.errChan <- fmt.Errorf(
						"could not execute order: [%v]",
						err,
					)
					return
				}

				if executed {
					p.manager.NotifyExecution(order)
					continue
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
