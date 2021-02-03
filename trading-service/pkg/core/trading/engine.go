package trading

import (
	"context"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/account"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/candle"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/logger"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/order"
	"sync"
	"time"
)

const (
	traderBackoff  = 10 * time.Second
	traderInterval = "1m"
)

type Pair struct {
	Base, Quote string
}

func (p Pair) String() string {
	return p.Base + p.Quote
}

type Exchange interface {
	account.Provider
	candle.Provider
	order.Submitter

	Name() string
}

type Engine struct {
	exchange Exchange

	tradersMutex sync.Mutex
	traders      map[Pair]bool
}

func NewEngine(exchange Exchange) *Engine {
	return &Engine{
		exchange: exchange,
		traders:  make(map[Pair]bool),
	}
}

func (e *Engine) ActivateTrader(ctx context.Context, pair Pair) {
	e.tradersMutex.Lock()
	defer e.tradersMutex.Unlock()

	logger := e.newContextLogger(pair)

	if _, traderExists := e.traders[pair]; traderExists {
		logger.Warningf("trader is already active")
		return
	}

	logger.Infof("activating trader")

	e.traders[pair] = true

	go func() {
		defer func() {
			e.tradersMutex.Lock()
			defer e.tradersMutex.Unlock()

			logger.Infof("deactivating trader")

			delete(e.traders, pair)
		}()

		for {
			if ctx.Err() != nil {
				return
			}

			e.runTraderInstance(ctx, logger, pair)

			time.Sleep(traderBackoff)
		}
	}()
}

func (e *Engine) newContextLogger(pair Pair) logger.Logger {
	return logger.WithFields(
		map[string]interface{}{
			"exchange": e.exchange.Name(),
			"pair":     pair,
			"interval": traderInterval,
		},
	)
}

func (e *Engine) ActiveTraders() int {
	e.tradersMutex.Lock()
	defer e.tradersMutex.Unlock()

	return len(e.traders)
}

func (e *Engine) runTraderInstance(
	ctx context.Context,
	logger logger.Logger,
	pair Pair,
) {
	logger.Infof("running trader instance")
	defer logger.Infof("terminating trader instance")

	traderCtx, cancelTraderCtx := context.WithCancel(ctx)
	defer cancelTraderCtx()

	logger.Infof("creating account manager")

	accountManager := account.NewManager(e.exchange, pair.Quote)

	now := time.Now()

	filter := &candle.Filter{
		Pair:      pair.String(),
		Interval:  traderInterval,
		StartTime: now.Add(-12 * time.Hour), // TODO: extend to 24h
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
		traderCtx,
		logger,
		e.exchange,
		filter,
		candleRegistry,
	)

	logger.Infof("creating order generator")

	orderGenerator := order.NewGenerator(
		logger,
		candleRegistry,
		accountManager,
	)

	logger.Infof("creating order registry")

	orderRegistry := order.NewRegistry()

	logger.Infof("running order executor")

	orderExecutor := order.RunExecutor(
		traderCtx,
		logger,
		orderGenerator,
		orderRegistry,
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
		case err := <-orderExecutor.ErrChan():
			logger.Errorf(
				"order executor error: [%v]",
				err,
			)
			return
		case <-traderCtx.Done():
			logger.Infof("trader context is done")
			return
		}
	}
}
