package binance

import (
	"context"
	"fmt"
	"github.com/adshao/go-binance"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/candle"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/trade"
	"math/big"
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
	return "BINANCE"
}

func (c *Client) Candles(
	ctx context.Context,
	filter *candle.Filter,
) ([]*candle.Candle, error) {
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

	candles := make([]*candle.Candle, len(klines))
	for index := range candles {
		kline := klines[index]

		candles[index] = &candle.Candle{
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
	filter *candle.Filter,
) (<-chan *candle.Tick, <-chan error) {
	tickChannel := make(chan *candle.Tick)
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

func (c *Client) parseKlineEvent(event *binance.WsKlineEvent) *candle.Tick {
	return &candle.Tick{
		Candle: &candle.Candle{
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

func (c *Client) ExecuteOrder(order *trade.Order) error {
	// TODO: implementation
	return nil
}

func (c *Client) IsOrderExecuted(order *trade.Order) (bool, error) {
	// TODO: implementation
	return false, nil
}

func (c *Client) AccountBalance(
	ctx context.Context,
	asset string,
) (*big.Float, error) {
	account, err := c.delegate.NewGetAccountService().Do(ctx)
	if err != nil {
		return nil, err
	}

	for _, balance := range account.Balances {
		if balance.Asset == asset {
			amount, ok := new(big.Float).SetString(balance.Free)
			if !ok {
				return nil, fmt.Errorf(
					"could not parse balance for asset [%v]",
					balance.Asset,
				)
			}

			return amount, nil
		}
	}

	return big.NewFloat(0), nil
}

func parseMilliseconds(milliseconds int64) time.Time {
	return time.Unix(0, milliseconds*int64(time.Millisecond))
}
