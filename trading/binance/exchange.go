package binance

import (
	"context"
	"github.com/adshao/go-binance"
	"github.com/lukasz-zimnoch/dexly/trading"
	"time"
)

const requestTimeout = 1 * time.Minute

type ExchangeService struct {
	client       *binance.Client
	exchangeInfo *binance.ExchangeInfo
	workload     *trading.Workload
}

func NewExchangeService(
	ctx context.Context,
	workload *trading.Workload,
) (*ExchangeService, error) {
	client := binance.NewClient(
		workload.Account.ExchangeApiKey,
		workload.Account.ExchangeSecretKey,
	)

	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	exchangeInfo, err := client.NewExchangeInfoService().Do(requestCtx)
	if err != nil {
		return nil, err
	}

	return &ExchangeService{
		client:       client,
		exchangeInfo: exchangeInfo,
		workload:     workload,
	}, nil
}

func (es *ExchangeService) Workload() *trading.Workload {
	return es.workload
}

func parseMilliseconds(milliseconds int64) time.Time {
	return time.Unix(0, milliseconds*int64(time.Millisecond))
}
