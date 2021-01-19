package trading

import (
	"context"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/candle"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/order"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/strategy"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	traderBackoff  = 10 * time.Second
	traderInterval = "1m"
)

type Exchange interface {
	candle.Provider
	order.Submitter

	Name() string
}

type Engine struct {
	exchange Exchange

	tradersMutex sync.Mutex
	traders      map[string]bool
}

func NewEngine(exchange Exchange) *Engine {
	return &Engine{
		exchange: exchange,
		traders:  make(map[string]bool),
	}
}

func (e *Engine) ActivateTrader(ctx context.Context, pair string) {
	e.tradersMutex.Lock()
	defer e.tradersMutex.Unlock()

	contextLogger := log.WithFields(
		log.Fields{
			"exchange": e.exchange.Name(),
			"pair":     pair,
			"interval": traderInterval,
		},
	)

	if _, traderExists := e.traders[pair]; traderExists {
		contextLogger.Warningf("trader is already active")
		return
	}

	contextLogger.Infof("activating trader")

	e.traders[pair] = true

	go func() {
		defer func() {
			e.tradersMutex.Lock()
			defer e.tradersMutex.Unlock()

			contextLogger.Infof("deactivating trader")

			delete(e.traders, pair)
		}()

		for {
			if ctx.Err() != nil {
				return
			}

			e.runTraderInstance(ctx, pair)

			time.Sleep(traderBackoff)
		}
	}()
}

func (e *Engine) ActiveTraders() int {
	e.tradersMutex.Lock()
	defer e.tradersMutex.Unlock()

	return len(e.traders)
}

func (e *Engine) runTraderInstance(ctx context.Context, pair string) {
	traderCtx, cancelTraderCtx := context.WithCancel(ctx)
	defer cancelTraderCtx()

	now := time.Now()

	filter := &candle.Filter{
		Pair:      pair,
		Interval:  traderInterval,
		StartTime: now.Add(-12 * time.Hour), // TODO: extend to 24h
		EndTime:   now,
	}

	contextLogger := log.WithFields(
		log.Fields{
			"exchange": e.exchange.Name(),
			"pair":     filter.Pair,
			"interval": filter.Interval,
		},
	)

	contextLogger.Infof("running trader instance")
	defer contextLogger.Infof("terminating trader instance")

	candleRegistrySize := int(filter.EndTime.Sub(filter.StartTime).Minutes())

	contextLogger.Infof(
		"creating candle registry with size [%v]",
		candleRegistrySize,
	)

	candleRegistry := candle.NewRegistry(candleRegistrySize)

	contextLogger.Infof("running candle monitor")

	candleMonitor := candle.RunMonitor(
		traderCtx,
		e.exchange,
		filter,
		candleRegistry,
	)

	contextLogger.Infof("creating strategy")

	strategyInstance := strategy.New(candleRegistry)

	contextLogger.Infof("creating order registry")

	orderRegistry := order.NewRegistry()

	contextLogger.Infof("running order executor")

	orderExecutor := order.RunExecutor(
		traderCtx,
		strategyInstance,
		orderRegistry,
		e.exchange,
	)

	for {
		select {
		case err := <-candleMonitor.ErrorChannel:
			contextLogger.Errorf(
				"trader detected candle monitor error: [%v]",
				err,
			)
			return
		case err := <-orderExecutor.ErrorChannel:
			contextLogger.Errorf(
				"trader detected order executor error: [%v]",
				err,
			)
			return
		case <-traderCtx.Done():
			contextLogger.Infof("trader context is done")
			return
		}
	}
}
