package daemon

import (
	"context"
	"github.com/google/uuid"
	"github.com/lukasz-zimnoch/dexly/trading"
	"sync"
	"time"
)

const workerRestartBackoff = 10 * time.Second

type CandleRepositoryFactoryFn func(windowSize int) trading.CandleRepository

type SignalGeneratorFactoryFn func(
	logger trading.Logger,
	pair trading.Pair,
	candleRepository trading.CandleRepository,
) trading.SignalGenerator

type WorkerController struct {
	logger                  trading.Logger
	account                 *trading.Account
	candleRepositoryFactory CandleRepositoryFactoryFn
	signalGeneratorFactory  SignalGeneratorFactoryFn
	positionRepository      trading.PositionRepository
	orderRepository         trading.OrderRepository
	exchange                trading.ExchangeService

	workersMutex   sync.Mutex
	workers        map[trading.Pair]bool
	workerInterval string // TODO: make it configurable
}

func RunWorkerController(
	logger trading.Logger,
	accountRepository trading.AccountRepository,
	candleRepositoryFactory CandleRepositoryFactoryFn,
	signalGeneratorFactory SignalGeneratorFactoryFn,
	positionRepository trading.PositionRepository,
	orderRepository trading.OrderRepository,
	exchange trading.ExchangeService,
) (*WorkerController, error) {
	// TODO: id doesn't matter for now
	account, err := accountRepository.Account(uuid.New())
	if err != nil {
		return nil, err
	}

	return &WorkerController{
		logger:                  logger,
		account:                 account,
		candleRepositoryFactory: candleRepositoryFactory,
		signalGeneratorFactory:  signalGeneratorFactory,
		positionRepository:      positionRepository,
		orderRepository:         orderRepository,
		exchange:                exchange,
		workers:                 make(map[trading.Pair]bool),
		workerInterval:          "1m",
	}, nil
}

func (wc *WorkerController) ActivateWorker(
	ctx context.Context,
	pair trading.Pair,
) {
	wc.workersMutex.Lock()
	defer wc.workersMutex.Unlock()

	workerLogger := wc.logger.WithFields(
		map[string]interface{}{
			"exchange": wc.exchange.ExchangeName(),
			"pair":     pair.String(),
			"interval": wc.workerInterval,
		},
	)

	if _, exists := wc.workers[pair]; exists {
		workerLogger.Warningf("worker is already active")
		return
	}

	workerLogger.Infof("activating worker")

	wc.workers[pair] = true

	go func() {
		defer func() {
			wc.workersMutex.Lock()
			defer wc.workersMutex.Unlock()

			workerLogger.Infof("deactivating worker")

			delete(wc.workers, pair)
		}()

		for {
			if ctx.Err() != nil {
				return
			}

			wc.runWorker(ctx, workerLogger, pair)

			time.Sleep(workerRestartBackoff)
		}
	}()
}

func (wc *WorkerController) ActiveWorkers() int {
	wc.workersMutex.Lock()
	defer wc.workersMutex.Unlock()

	return len(wc.workers)
}

func (wc *WorkerController) runWorker(
	ctx context.Context,
	workerLogger trading.Logger,
	pair trading.Pair,
) {
	workerLogger.Infof("running worker")
	defer workerLogger.Infof("terminating worker")

	workerCtx, cancelWorkerCtx := context.WithCancel(ctx)
	defer cancelWorkerCtx()

	now := time.Now()

	filter := &trading.CandleFilter{
		Pair:      pair.String(),
		Interval:  wc.workerInterval,
		StartTime: now.Add(-12 * time.Hour),
		EndTime:   now,
	}

	candleRegistrySize := int(filter.EndTime.Sub(filter.StartTime).Minutes())

	workerLogger.Infof(
		"creating candle repository with size [%v]",
		candleRegistrySize,
	)

	candleRepository := wc.candleRepositoryFactory(candleRegistrySize)

	workerLogger.Infof("running candle monitor")

	candleMonitor := RunCandleMonitor(
		workerCtx,
		workerLogger,
		wc.exchange,
		filter,
		candleRepository,
	)

	workerLogger.Infof("running trading worker")

	worker := trading.RunWorker(
		workerCtx,
		workerLogger,
		wc.account,
		pair,
		wc.signalGeneratorFactory(workerLogger, pair, candleRepository),
		candleRepository,
		wc.positionRepository,
		wc.orderRepository,
		wc.exchange,
	)

	for {
		select {
		case err := <-candleMonitor.ErrChan():
			workerLogger.Errorf(
				"candle monitor error: [%v]",
				err,
			)
			return
		case err := <-worker.ErrChan():
			workerLogger.Errorf(
				"worker error: [%v]",
				err,
			)
			return
		case <-workerCtx.Done():
			workerLogger.Infof("worker context is done")
			return
		}
	}
}
