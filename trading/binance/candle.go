package binance

import (
	"context"
	"github.com/adshao/go-binance"
	"github.com/lukasz-zimnoch/dexly/trading"
)

func (es *ExchangeService) Candles(
	ctx context.Context,
	filter *trading.CandleFilter,
) ([]*trading.Candle, error) {
	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	klines, err := es.client.
		NewKlinesService().
		Symbol(filter.Pair).
		Interval(filter.Interval).
		StartTime(filter.StartTime.UnixNano() / 1e6).
		EndTime(filter.EndTime.UnixNano() / 1e6).
		Limit(1000).
		Do(requestCtx)
	if err != nil {
		return nil, err
	}

	candles := make([]*trading.Candle, len(klines))
	for index := range candles {
		kline := klines[index]

		candles[index] = &trading.Candle{
			Pair:       filter.Pair,
			Exchange:   es.ExchangeName(),
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

func (es *ExchangeService) CandlesTicker(
	ctx context.Context,
	filter *trading.CandleFilter,
) (<-chan *trading.CandleTick, <-chan error) {
	tickChannel := make(chan *trading.CandleTick)
	errorChannel := make(chan error)

	go func() {
		_, stopChannel, err := binance.WsKlineServe(
			filter.Pair,
			filter.Interval,
			func(event *binance.WsKlineEvent) {
				tickChannel <- es.parseKlineEvent(event)
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

func (es *ExchangeService) parseKlineEvent(
	event *binance.WsKlineEvent,
) *trading.CandleTick {
	return &trading.CandleTick{
		Candle: &trading.Candle{
			Pair:       event.Symbol,
			Exchange:   es.ExchangeName(),
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
