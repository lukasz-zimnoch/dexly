package binance

import (
	"context"
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading"
	"math/big"
)

func (es *ExchangeService) AccountBalances(
	ctx context.Context,
) (trading.Balances, error) {
	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	account, err := es.client.NewGetAccountService().Do(requestCtx)
	if err != nil {
		return nil, err
	}

	balances := make(map[trading.Asset]*big.Float)

	for _, balance := range account.Balances {
		amount, ok := new(big.Float).SetString(balance.Free)
		if !ok {
			return nil, fmt.Errorf(
				"could not parse balance for asset [%v]",
				balance.Asset,
			)
		}

		if amount.Cmp(big.NewFloat(0)) == 0 {
			continue
		}

		balances[trading.Asset(balance.Asset)] = amount
	}

	return balances, nil
}

func (es *ExchangeService) AccountTakerCommission(
	ctx context.Context,
) (*big.Float, error) {
	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	account, err := es.client.NewGetAccountService().Do(requestCtx)
	if err != nil {
		return nil, err
	}

	return big.NewFloat(float64(account.TakerCommission / 10000)), nil
}
