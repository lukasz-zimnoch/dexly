package binance

import (
	"context"
	"fmt"
	"github.com/adshao/go-binance"
	"github.com/adshao/go-binance/common"
	"github.com/lukasz-zimnoch/dexly/trading"
)

func (es *ExchangeService) ExecuteOrder(
	ctx context.Context,
	order *trading.Order,
) (bool, error) {
	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	symbol := string(es.workload.Pair.Symbol())
	symbolInfo, ok := es.findSymbolInfo(symbol)
	if !ok {
		return false, fmt.Errorf("could not find info for symbol: [%v]", symbol)
	}

	response, err := es.client.NewCreateOrderService().
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

func (es *ExchangeService) findSymbolInfo(
	symbol string,
) (*binance.Symbol, bool) {
	for _, symbolInfo := range es.exchangeInfo.Symbols {
		if symbolInfo.Symbol == symbol {
			return &symbolInfo, true
		}
	}

	return nil, false
}

func (es *ExchangeService) IsOrderExecuted(
	ctx context.Context,
	order *trading.Order,
) (bool, error) {
	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	response, err := es.client.NewGetOrderService().
		Symbol(string(es.workload.Pair.Symbol())).
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
