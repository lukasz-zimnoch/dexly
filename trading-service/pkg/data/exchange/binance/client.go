package binance

import (
	"context"
	"github.com/adshao/go-binance"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core"
	"time"
)

type Client struct {
	delegate *binance.Client
}

func NewClient(apiKey, secretKey string) *Client {
	return &Client{
		delegate: binance.NewClient(apiKey, secretKey),
	}
}

func (c *Client) Name() string {
	return "binance"
}

func (c *Client) Candles(
	ctx context.Context,
	filter *core.CandlesFilter,
) ([]*core.Candle, error) {
	klines, err := c.delegate.
		NewKlinesService().
		Symbol(filter.Pair).
		Interval(filter.Interval).
		StartTime(filter.StartTime.UnixNano() / 1e6).
		EndTime(filter.EndTime.UnixNano() / 1e6).
		Limit(1000).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	candles := make([]*core.Candle, len(klines))
	for index := range candles {
		kline := klines[index]

		candles[index] = &core.Candle{
			Pair:       filter.Pair,
			Exchange:   c.Name(),
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

func (c *Client) CandlesTicker(
	ctx context.Context,
	filter *core.CandlesFilter,
) (<-chan *core.CandleTick, <-chan error) {
	tickChannel := make(chan *core.CandleTick)
	errorChannel := make(chan error)

	go func() {
		_, stopChannel, err := binance.WsKlineServe(
			filter.Pair,
			filter.Interval,
			func(event *binance.WsKlineEvent) {
				tickChannel <- c.parseKlineEvent(event)
			},
			func(err error) {
				errorChannel <- err
			},
		)
		if err != nil {
			errorChannel <- err
			return
		}

		<-ctx.Done()
		close(stopChannel)
	}()

	return tickChannel, errorChannel
}

func (c *Client) parseKlineEvent(event *binance.WsKlineEvent) *core.CandleTick {
	return &core.CandleTick{
		Candle: &core.Candle{
			Pair:       event.Symbol,
			Exchange:   c.Name(),
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

func parseMilliseconds(milliseconds int64) time.Time {
	return time.Unix(0, milliseconds*int64(time.Millisecond))
}
