package core

import (
	"context"
	log "github.com/sirupsen/logrus"
	"time"
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
}

func NewTradingEngine(exchange ExchangeClient) *TradingEngine {
	return &TradingEngine{
		exchange: exchange,
	}
}

func (te *TradingEngine) RunTrading(ctx context.Context, pair string) {
	tradingCtx, cancelTradingCtx := context.WithCancel(ctx)
	defer cancelTradingCtx()

	now := time.Now()

	filter := &CandleFilter{
		Pair:      pair,
		Interval:  "1m",
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

	contextLogger.Infof("starting trading engine")
	defer contextLogger.Infof("terminating trading engine")

	analyser := newAnalyser()

	candles, err := te.exchange.Candles(tradingCtx, filter)
	if err != nil {
		contextLogger.Errorf(
			"trading engine failed to get candles: [%v]",
			err,
		)
		return
	}

	contextLogger.Debugf(
		"trading engine fetched [%v] historical candles",
		len(candles),
	)

	for _, candle := range candles {
		analyser.addCandle(candle)
	}

	candlesTicker, err := te.exchange.CandlesTicker(tradingCtx, filter)
	if err != nil {
		contextLogger.Errorf(
			"trading engine failed to get candles ticker: [%v]",
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
					"trading engine detected candles ticker termination",
				)
				cancelTradingCtx()
				continue
			}

			analyser.addCandle(candleTick.Candle)

			if !tickTimeoutTimer.Stop() {
				<-tickTimeoutTimer.C
			}
			tickTimeoutTimer.Reset(tickTimeout)
		case <-tickTimeoutTimer.C:
			contextLogger.Errorf(
				"trading engine detected candle tick timeout expiration",
			)
			cancelTradingCtx()
		case <-tradingCtx.Done():
			contextLogger.Infof("trading engine context is done")
			return
		}
	}
}
