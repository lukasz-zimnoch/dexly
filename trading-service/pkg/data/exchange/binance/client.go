package binance

import (
	"context"
	"fmt"
	"github.com/adshao/go-binance"
	"github.com/adshao/go-binance/common"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/candle"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/trade"
	"math/big"
	"time"
)

const requestTimeout = 1 * time.Minute

type Client struct {
	delegate     *binance.Client
	exchangeInfo *binance.ExchangeInfo
}

func NewClient(
	ctx context.Context,
	apiKey,
	secretKey string,
	testnet bool,
) (*Client, error) {
	binanceClient := binance.NewClient(apiKey, secretKey)

	if testnet {
		binanceClient.BaseURL = "https://testnet.binance.vision"
	}

	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	exchangeInfo, err := binanceClient.NewExchangeInfoService().Do(requestCtx)
	if err != nil {
		return nil, err
	}

	return &Client{
		delegate:     binanceClient,
		exchangeInfo: exchangeInfo,
	}, nil
}

func (c *Client) Name() string {
	return "BINANCE"
}

func (c *Client) Candles(
	ctx context.Context,
	filter *candle.Filter,
) ([]*candle.Candle, error) {
	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	klines, err := c.delegate.
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

func (c *Client) ExecuteOrder(
	ctx context.Context,
	order *trade.Order,
) (bool, error) {
	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	symbol := order.Position.Pair
	symbolInfo, ok := c.findSymbolInfo(symbol)
	if !ok {
		return false, fmt.Errorf("could not find info for symbol: %v", symbol)
	}

	response, err := c.delegate.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideType(order.Side.String())).
		Type(binance.OrderTypeLimit).
		NewClientOrderID(order.ID.String()).
		Price(order.Price.Text('f', symbolInfo.QuotePrecision)).
		Quantity(order.Size.Text('f', symbolInfo.BaseAssetPrecision)).
		// fill or kill (FOK) orders are either filled immediately or cancelled
		TimeInForce(binance.TimeInForceTypeFOK).
		Do(requestCtx)
	if err != nil {
		// Request error - return it to the caller.
		return false, err
	}

	if response.Status != binance.OrderStatusTypeFilled {
		// The order's status is other than FILLED. Because the order is FOK,
		// it has been probably cancelled. Return that info to the caller
		// but it's not an error situation.
		return false, nil
	}

	// Everything good, the order was FILLED.
	return true, nil
}

func (c *Client) IsOrderExecuted(
	ctx context.Context,
	order *trade.Order,
) (bool, error) {
	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	response, err := c.delegate.NewGetOrderService().
		Symbol(order.Position.Pair).
		OrigClientOrderID(order.ID.String()).
		Do(requestCtx)
	if err != nil {
		if common.IsAPIError(err) {
			apiErr := err.(*common.APIError)
			// -2013 is the code of NO_SUCH_ORDER error according to the docs
			// https://binance-docs.github.io/apidocs/spot/en/#error-codes
			if apiErr.Code == -2013 {
				// Given order doesn't exist so we are returning false to
				// the caller but it's not an error situation.
				return false, nil
			}
		}

		// Other request error - return it to the caller
		return false, err
	}

	// We send FOK orders so an executed order will always have FILLED status.
	return response.Status == binance.OrderStatusTypeFilled, nil
}

func (c *Client) AccountBalance(
	ctx context.Context,
	asset string,
) (*big.Float, error) {
	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	account, err := c.delegate.NewGetAccountService().Do(requestCtx)
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

func (c *Client) AccountTakerCommission(
	ctx context.Context,
) (*big.Float, error) {
	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	account, err := c.delegate.NewGetAccountService().Do(requestCtx)
	if err != nil {
		return nil, err
	}

	return big.NewFloat(float64(account.TakerCommission / 10000)), nil
}

func (c *Client) findSymbolInfo(symbol string) (*binance.Symbol, bool) {
	for _, symbolInfo := range c.exchangeInfo.Symbols {
		if symbolInfo.Symbol == symbol {
			return &symbolInfo, true
		}
	}

	return nil, false
}

func parseMilliseconds(milliseconds int64) time.Time {
	return time.Unix(0, milliseconds*int64(time.Millisecond))
}
