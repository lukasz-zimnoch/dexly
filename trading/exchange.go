package trading

import (
	"context"
	"math/big"
	"time"
)

type ExchangeConnector interface {
	Connect(ctx context.Context, workload *Workload) (ExchangeService, error)
}

type ExchangeService interface {
	ExchangeCandleService
	ExchangeAccountService
	ExchangeOrderService

	Workload() *Workload
}

type ExchangeCandleService interface {
	Candles(ctx context.Context, start, end time.Time) ([]*Candle, error)

	CandlesTicker(ctx context.Context) (<-chan *CandleTick, <-chan error)
}

type ExchangeAccountService interface {
	AccountTakerCommission(ctx context.Context) (*big.Float, error)

	AccountBalances(ctx context.Context) (Balances, error)
}

type ExchangeOrderService interface {
	ExecuteOrder(ctx context.Context, order *Order) (bool, error)

	IsOrderExecuted(ctx context.Context, order *Order) (bool, error)
}
