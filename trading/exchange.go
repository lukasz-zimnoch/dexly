package trading

import (
	"context"
	"math/big"
)

type ExchangeCandleService interface {
	Candles(ctx context.Context, filter *CandleFilter) ([]*Candle, error)

	CandlesTicker(
		ctx context.Context,
		filter *CandleFilter,
	) (<-chan *CandleTick, <-chan error)
}

type ExchangeAccountService interface {
	ExchangeAccount(
		ctx context.Context,
		account *Account,
	) (*ExchangeAccount, error)
}

type ExchangeOrderService interface {
	ExecuteOrder(ctx context.Context, order *Order) (bool, error)

	IsOrderExecuted(ctx context.Context, order *Order) (bool, error)
}

type ExchangeService interface {
	ExchangeCandleService
	ExchangeOrderService
	ExchangeAccountService

	ExchangeName() string
}

type ExchangeAccount struct {
	*Account

	Exchange        string
	Balances        map[Asset]*big.Float
	TakerCommission *big.Float
}

func (ea *ExchangeAccount) AssetBalance(asset Asset) *big.Float {
	for balanceAsset, balanceValue := range ea.Balances {
		if balanceAsset == asset {
			return balanceValue
		}
	}

	return big.NewFloat(0)
}
