package binance

import (
	"context"
	"fmt"
	"github.com/adshao/go-binance"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core"
	"time"
)

// TODO: extract those constants
const (
	exchangeName = "binance"
	interval     = "1m"
	apiKey       = "?"
	secretKey    = "?"
)

type Client struct {
	delegate *binance.Client
}

func NewClient() *Client {
	return &Client{
		delegate: binance.NewClient(apiKey, secretKey),
	}
}

func (c *Client) CandlesTicker(
	ctx context.Context,
	symbol string,
) (chan *core.CandleTick, error) {
	tickChannel := make(chan *core.CandleTick) // TODO: should be buffered?

	eventHandler := func(event *binance.WsKlineEvent) {
		tickChannel <- &core.CandleTick{
			Candle: &core.Candle{
				Symbol:     event.Symbol,
				Exchange:   exchangeName,
				OpenTime:   parseMilliseconds(event.Kline.StartTime),
				CloseTime:  parseMilliseconds(event.Kline.EndTime),
				OpenPrice:  event.Kline.Open,
				ClosePrice: event.Kline.Close,
				MaxPrice:   event.Kline.High,
				MinPrice:   event.Kline.Low,
				Volume:     event.Kline.Volume,
				TradeCount: uint(event.Kline.TradeNum),
			},
			TickTime: parseMilliseconds(event.Time),
		}
	}

	errorHandler := func(err error) {
		fmt.Printf("received error: [%v]", err)
	}

	_, stopChannel, err := binance.WsKlineServe(
		symbol,
		interval,
		eventHandler,
		errorHandler,
	)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		stopChannel <- struct{}{}
	}()

	return tickChannel, nil
}

func (c *Client) Candles(
	ctx context.Context,
	symbol string,
) ([]*core.Candle, error) {
	klines, err := c.delegate.
		NewKlinesService().
		Symbol(symbol).
		Interval(interval).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	candles := make([]*core.Candle, len(klines))
	for index := range candles {
		kline := klines[index]

		candles[index] = &core.Candle{
			Symbol:     symbol,
			Exchange:   exchangeName,
			OpenTime:   parseMilliseconds(kline.OpenTime),
			CloseTime:  parseMilliseconds(kline.CloseTime),
			OpenPrice:  kline.Open,
			ClosePrice: kline.Close,
			MaxPrice:   kline.High,
			MinPrice:   kline.Low,
			Volume:     kline.Volume,
			TradeCount: uint(kline.TradeNum),
		}
	}

	return candles, nil
}

func parseMilliseconds(milliseconds int64) time.Time {
	return time.Unix(0, milliseconds*int64(time.Millisecond))
}
