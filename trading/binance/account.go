package binance

import (
	"context"
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading"
	"math/big"
)

func (es *ExchangeService) ExchangeAccount(
	ctx context.Context,
	account *trading.Account,
) (*trading.ExchangeAccount, error) {
	balances, err := es.accountBalances(ctx)
	if err != nil {
		return nil, err
	}

	takerCommission, err := es.accountTakerCommission(ctx)
	if err != nil {
		return nil, err
	}

	return &trading.ExchangeAccount{
		Account:         account,
		Exchange:        es.ExchangeName(),
		Balances:        balances,
		TakerCommission: takerCommission,
	}, nil
}

func (es *ExchangeService) accountBalances(
	ctx context.Context,
) (map[trading.Asset]*big.Float, error) {
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

func (es *ExchangeService) accountTakerCommission(
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
