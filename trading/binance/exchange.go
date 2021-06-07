package binance

import (
	"context"
	"github.com/adshao/go-binance"
	"time"
)

const requestTimeout = 1 * time.Minute

type ExchangeService struct {
	client       *binance.Client
	exchangeInfo *binance.ExchangeInfo
}

func NewExchangeService(
	ctx context.Context,
	apiKey,
	secretKey string,
	testnet bool,
) (*ExchangeService, error) {
	client := binance.NewClient(apiKey, secretKey)

	if testnet {
		client.BaseURL = "https://testnet.binance.vision"
	}

	requestCtx, cancelRequestCtx := context.WithTimeout(ctx, requestTimeout)
	defer cancelRequestCtx()

	exchangeInfo, err := client.NewExchangeInfoService().Do(requestCtx)
	if err != nil {
		return nil, err
	}

	return &ExchangeService{
		client:       client,
		exchangeInfo: exchangeInfo,
	}, nil
}

func (es *ExchangeService) ExchangeName() string {
	return "BINANCE"
}

func parseMilliseconds(milliseconds int64) time.Time {
	return time.Unix(0, milliseconds*int64(time.Millisecond))
}
