package core

import (
	"context"
	"fmt"
)

type ExchangeClient interface {
	CandlesTicker(ctx context.Context, symbol string) (chan *CandleTick, error)
	Candles(ctx context.Context, symbol string) ([]*Candle, error)
}

type TradingEngine struct {
	exchange ExchangeClient
}

func NewTradingEngine(exchange ExchangeClient) *TradingEngine {
	return &TradingEngine{exchange}
}

func (te *TradingEngine) Observe(ctx context.Context, symbol string) {
	analyser := newAnalyser()

	candles, err := te.exchange.Candles(ctx, symbol)
	if err != nil {
		fmt.Println(err)
	}

	for _, candle := range candles {
		analyser.addCandle(candle)
	}

	candlesTicker, err := te.exchange.CandlesTicker(ctx, symbol)
	if err != nil {
		fmt.Println(err)
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
