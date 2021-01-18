package core

import (
	"context"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	traderBackoff  = 10 * time.Second
	traderInterval = "1m"
)

type Candle struct {
	Pair       string
	Exchange   string
	OpenTime   time.Time
	CloseTime  time.Time
	OpenPrice  string
	ClosePrice string
	MaxPrice   string
	MinPrice   string
	Volume     string
	TradeCount uint
}

func (c *Candle) Equal(other *Candle) bool {
	return c.OpenTime.Equal(other.OpenTime) &&
		c.CloseTime.Equal(other.CloseTime)
}

type CandleTick struct {
	*Candle
	TickTime time.Time
}

type CandlesFilter struct {
	Pair      string
	Interval  string
	StartTime time.Time
	EndTime   time.Time
}

type ExchangeClient interface {
	Name() string

	Candles(
		ctx context.Context,
		filter *CandlesFilter,
	) ([]*Candle, error)

	CandlesTicker(
		ctx context.Context,
		filter *CandlesFilter,
	) (<-chan *CandleTick, <-chan error)
}

type TradingEngine struct {
	exchange ExchangeClient

	traders      map[string]bool
	tradersMutex sync.Mutex
}

func NewTradingEngine(exchange ExchangeClient) *TradingEngine {
	return &TradingEngine{
		exchange: exchange,
		traders:  make(map[string]bool),
	}
}

func (te *TradingEngine) ActivateTrader(ctx context.Context, pair string) {
	te.tradersMutex.Lock()
	defer te.tradersMutex.Unlock()

	contextLogger := log.WithFields(
		log.Fields{
			"exchange": te.exchange.Name(),
			"pair":     pair,
			"interval": traderInterval,
		},
	)

	if _, traderExists := te.traders[pair]; traderExists {
		contextLogger.Warningf("trader is already active")
		return
	}

	contextLogger.Infof("activating trader")

	te.traders[pair] = true

	go func() {
		defer func() {
			te.tradersMutex.Lock()
			defer te.tradersMutex.Unlock()

			contextLogger.Infof("deactivating trader")

			delete(te.traders, pair)
		}()

		for {
			if ctx.Err() != nil {
				return
			}

			te.runTraderInstance(ctx, pair)

			time.Sleep(traderBackoff)
		}
	}()
}

func (te *TradingEngine) ActiveTraders() int {
	te.tradersMutex.Lock()
	defer te.tradersMutex.Unlock()

	return len(te.traders)
}

func (te *TradingEngine) runTraderInstance(ctx context.Context, pair string) {
	traderCtx, cancelTraderCtx := context.WithCancel(ctx)
	defer cancelTraderCtx()

	now := time.Now()

	filter := &CandlesFilter{
		Pair:      pair,
		Interval:  traderInterval,
		StartTime: now.Add(-12 * time.Hour), // TODO: extend to 24h
		EndTime:   now,
	}

	contextLogger := log.WithFields(
		log.Fields{
			"exchange": te.exchange.Name(),
			"pair":     filter.Pair,
			"interval": filter.Interval,
		},
	)

	contextLogger.Infof("running trader instance")
	defer contextLogger.Infof("terminating trader instance")

	candlesRegistrySize := int(filter.EndTime.Sub(filter.StartTime).Minutes())
	candlesRegistry := newCandlesRegistry(candlesRegistrySize)

	candlesMonitor := newCandlesMonitor(te.exchange)
	candlesMonitorErrorChannel := candlesMonitor.run(traderCtx, filter, candlesRegistry)

	positionManager := newPositionManager()
	orderRequestChannel := positionManager.run(traderCtx, candlesRegistry)

	orderExecutor := newOrderExecutor(te.exchange)
	orderExecutorChannel := orderExecutor.run(traderCtx)

	for {
		select {
		case orderRequest := <-orderRequestChannel:
			contextLogger.Infof("trader detected order request")

			select {
			case orderExecutorChannel <- orderRequest:
			default:
				contextLogger.Warningf("order executor is busy")
			}
		case err := <-candlesMonitorErrorChannel:
			contextLogger.Errorf(
				"trader detected candles monitor error: [%v]",
				err,
			)
			return
		case <-traderCtx.Done():
			contextLogger.Infof("trader context is done")
			return
		}
	}
}
