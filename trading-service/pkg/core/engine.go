package core

import (
	"context"
	log "github.com/sirupsen/logrus"
	"time"
)

const traderBackoff = 10 * time.Second

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
	// TODO: prevent multiple traders on same pair

	go func() {
		for {
			if ctx.Err() != nil {
				return
			}

			te.runTrader(ctx, pair)
			time.Sleep(traderBackoff)
		}
	}()
}

func (te *TradingEngine) runTrader(ctx context.Context, pair string) {
	traderCtx, cancelTraderCtx := context.WithCancel(ctx)
	defer cancelTraderCtx()

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

	contextLogger.Infof("starting trader")
	defer contextLogger.Infof("terminating trader")

	analyser := newAnalyser()

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

	for _, candle := range candles {
		analyser.addCandle(candle)
	}

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

			analyser.addCandle(candleTick.Candle)

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
