package core

import (
	"context"
	log "github.com/sirupsen/logrus"
	"time"
)

type CandleFilter struct {
	Symbol    string
	Interval  string
	StartTime time.Time
	EndTime   time.Time
}

type ExchangeClient interface {
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
	return &TradingEngine{exchange}
}

func (te *TradingEngine) Observe(ctx context.Context, symbol string) {
	analyser := newAnalyser()

	now := time.Now()

	filter := &CandleFilter{
		Symbol:    symbol,
		Interval:  "1m",
		StartTime: now.Add(-12 * time.Hour), // TODO: extend to 24h
		EndTime:   now,
	}

	contextLogger := log.WithFields(
		log.Fields{
			"symbol":    filter.Symbol,
			"interval":  filter.Interval,
			"startTime": filter.StartTime.Format(time.RFC3339),
			"endTime":   filter.EndTime.Format(time.RFC3339),
		},
	)

	candles, err := te.exchange.Candles(ctx, filter)
	if err != nil {
		contextLogger.Errorf("could not get candles: [%v]", err)
	}

	contextLogger.Debugf("fetched [%v] historical candles", len(candles))

	for _, candle := range candles {
		analyser.addCandle(candle)
	}

	candlesTicker, err := te.exchange.CandlesTicker(ctx, filter)
	if err != nil {
		contextLogger.Errorf("could not get candles ticker: [%v]", err)
	}

	for {
		select {
		case candleTick := <-candlesTicker:
			analyser.addCandle(candleTick.Candle)
		case <-ctx.Done():
			return
		}
	}
}
