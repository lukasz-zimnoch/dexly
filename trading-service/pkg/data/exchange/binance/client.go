package binance

import (
	"context"
	"github.com/adshao/go-binance"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core"
	log "github.com/sirupsen/logrus"
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

func (c *Client) CandlesTicker(
	ctx context.Context,
	filter *core.CandleFilter,
) (chan *core.CandleTick, error) {
	tickerCtx, cancelTickerCtx := context.WithCancel(ctx)

	contextLogger := log.WithFields(
		log.Fields{
			"exchange": c.Name(),
			"pair":     filter.Pair,
			"interval": filter.Interval,
		},
	)

	eventChannel := make(chan *binance.WsKlineEvent)

	eventHandler := func(event *binance.WsKlineEvent) {
		eventChannel <- event
	}

	errorHandler := func(err error) {
		contextLogger.Error("candles ticker received an error: [%v]", err)
		cancelTickerCtx()
	}

	doneChannel, stopChannel, err := binance.WsKlineServe(
		filter.Pair,
		filter.Interval,
		eventHandler,
		errorHandler,
	)
	if err != nil {
		cancelTickerCtx()
		return nil, err
	}

	tickChannel := make(chan *core.CandleTick)

	go func() {
		contextLogger.Infof("starting candles ticker")
		defer contextLogger.Infof("terminating candles ticker")

	eventLoop:
		for {
			select {
			case event := <-eventChannel:
				tickChannel <- c.parseKlineEvent(event)
			case <-doneChannel:
				contextLogger.Infof(
					"candles ticker connection has been terminated",
				)
				break eventLoop
			case <-tickerCtx.Done():
				contextLogger.Infof("candles ticker context is done")
				break eventLoop
			}
		}

		close(stopChannel) // stop the websocket connection if not done yet
		close(tickChannel) // notify clients about ticker termination
	}()

	return tickChannel, nil
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

func (c *Client) Candles(
	ctx context.Context,
	filter *core.CandleFilter,
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

func parseMilliseconds(milliseconds int64) time.Time {
	return time.Unix(0, milliseconds*int64(time.Millisecond))
}
