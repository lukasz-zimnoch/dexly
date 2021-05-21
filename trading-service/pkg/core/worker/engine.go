package worker

import (
	"context"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/account"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/candle"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/logger"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/strategy"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/trade"
	"sync"
	"time"
)

const (
	workerBackoff  = 10 * time.Second
	workerInterval = "1m"
)

type Pair struct {
	Base, Quote string
}

func (p Pair) String() string {
	return p.Base + p.Quote
}

type exchange interface {
	account.Supplier
	candle.Supplier
	executor

	Name() string
}

type Engine struct {
	exchange        exchange
	tradeRepository trade.Repository

	workersMutex sync.Mutex
	workers      map[Pair]bool
}

func NewEngine(exchange exchange, tradeRepository trade.Repository) *Engine {
	return &Engine{
		exchange:        exchange,
		tradeRepository: tradeRepository,
		workers:         make(map[Pair]bool),
	}
}

func (e *Engine) ActivateWorker(ctx context.Context, pair Pair) {
	e.workersMutex.Lock()
	defer e.workersMutex.Unlock()

	logger := e.newContextLogger(pair)

	if _, exists := e.workers[pair]; exists {
		logger.Warningf("worker is already active")
		return
	}

	logger.Infof("activating worker")

	e.workers[pair] = true

	go func() {
		defer func() {
			e.workersMutex.Lock()
			defer e.workersMutex.Unlock()

			logger.Infof("deactivating worker")

			delete(e.workers, pair)
		}()

		for {
			if ctx.Err() != nil {
				return
			}

			e.runWorker(ctx, logger, pair)

			time.Sleep(workerBackoff)
		}
	}()
}

func (e *Engine) newContextLogger(pair Pair) logger.Logger {
	return logger.WithFields(
		map[string]interface{}{
			"exchange": e.exchange.Name(),
			"pair":     pair.String(),
			"interval": workerInterval,
		},
	)
}

func (e *Engine) ActiveWorkers() int {
	e.workersMutex.Lock()
	defer e.workersMutex.Unlock()

	return len(e.workers)
}

func (e *Engine) runWorker(
	ctx context.Context,
	logger logger.Logger,
	pair Pair,
) {
	logger.Infof("running worker")
	defer logger.Infof("terminating worker")

	workerCtx, cancelWorkerCtx := context.WithCancel(ctx)
	defer cancelWorkerCtx()

	now := time.Now()

	filter := &candle.Filter{
		Pair:      pair.String(),
		Interval:  workerInterval,
		StartTime: now.Add(-12 * time.Hour),
		EndTime:   now,
	}

	candleRegistrySize := int(filter.EndTime.Sub(filter.StartTime).Minutes())

	logger.Infof(
		"creating candle registry with size [%v]",
		candleRegistrySize,
	)

	candleRegistry := candle.NewRegistry(candleRegistrySize)

	logger.Infof("running candle monitor")

	candleMonitor := candle.RunMonitor(
		workerCtx,
		logger,
		e.exchange,
		filter,
		candleRegistry,
	)

	logger.Infof("running trading pipeline")

	pipeline := runPipeline(
		workerCtx,
		logger,
		strategy.NewEmaCross(
			logger,
			candleRegistry,
		),
		trade.NewManager(
			logger,
			candleRegistry,
			account.NewManager(e.exchange, pair.Quote),
			e.tradeRepository,
			pair.String(),
			e.exchange.Name(),
		),
		e.exchange,
	)

	for {
		select {
		case err := <-candleMonitor.ErrChan():
			logger.Errorf(
				"candle monitor error: [%v]",
				err,
			)
			return
		case err := <-pipeline.errChan:
			logger.Errorf(
				"worker pipeline error: [%v]",
				err,
			)
			return
		case <-workerCtx.Done():
			logger.Infof("worker context is done")
			return
		}
	}
}
