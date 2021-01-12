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

type CandleFilter struct {
	Pair      string
	Interval  string
	StartTime time.Time
	EndTime   time.Time
}

type ExchangeClient interface {
	Name() string

	Candles(
		ctx context.Context,
		filter *CandleFilter,
	) ([]*Candle, error)

	CandlesTicker(
		ctx context.Context,
		filter *CandleFilter,
	) (chan *CandleTick, error)
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

	filter := &CandleFilter{
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

	candles, err := te.exchange.Candles(traderCtx, filter)
	if err != nil {
		contextLogger.Errorf(
			"trader failed to get candles: [%v]",
			err,
		)
		return
	}

	contextLogger.Debugf(
		"trader fetched [%v] historical candles",
		len(candles),
	)

	analyser := newAnalyser(len(candles))
	analyser.addCandles(candles...)

	candlesTicker, err := te.exchange.CandlesTicker(traderCtx, filter)
	if err != nil {
		contextLogger.Errorf(
			"trader failed to get candles ticker: [%v]",
			err,
		)
		return
	}

	tickTimeout := 10 * time.Second
	tickTimeoutTimer := time.NewTimer(tickTimeout)

	for {
		select {
		case candleTick, more := <-candlesTicker:
			if !more {
				contextLogger.Errorf(
					"trader detected candles ticker termination",
				)
				cancelTraderCtx()
				continue
			}

			contextLogger.Debugf(
				"trader received candle tick with "+
					"open time [%v] and close price [%v]",
				candleTick.OpenTime.Format(time.RFC3339),
				candleTick.ClosePrice,
			)

			analyser.addCandles(candleTick.Candle)

			if !tickTimeoutTimer.Stop() {
				<-tickTimeoutTimer.C
			}
			tickTimeoutTimer.Reset(tickTimeout)
		case <-tickTimeoutTimer.C:
			contextLogger.Errorf("trader detected tick timeout expiration")
			cancelTraderCtx()
		case <-traderCtx.Done():
			contextLogger.Infof("trader context is done")
			return
		}
	}
}
