package binance

import (
	"context"
	"github.com/adshao/go-binance"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core"
	log "github.com/sirupsen/logrus"
	"time"
)

const exchangeName = "binance"

type Client struct {
	delegate *binance.Client
}

func NewClient(apiKey, secretKey string) *Client {
	return &Client{
		delegate: binance.NewClient(apiKey, secretKey),
	}
}

func (c *Client) CandlesTicker(
	ctx context.Context,
	filter *core.CandleFilter,
) (chan *core.CandleTick, error) {
	contextLogger := log.WithFields(
		log.Fields{
			"exchange": exchangeName,
			"symbol":   filter.Symbol,
			"interval": filter.Interval,
		},
	)

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
		contextLogger.Error("error from candles ticker: [%v]", err)
	}

	_, stopChannel, err := binance.WsKlineServe(
		filter.Symbol,
		filter.Interval,
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
	filter *core.CandleFilter,
) ([]*core.Candle, error) {
	klines, err := c.delegate.
		NewKlinesService().
		Symbol(filter.Symbol).
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
			Symbol:     filter.Symbol,
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
