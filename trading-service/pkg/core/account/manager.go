package account

import (
	"context"
	"math/big"
	"time"
)

type Provider interface {
	AccountBalance(
		ctx context.Context,
		asset string,
	) (*big.Float, error)
}

type Manager struct {
	provider Provider
	asset    string
}

func NewManager(provider Provider, asset string) *Manager {
	return &Manager{
		provider: provider,
		asset:    asset,
	}
}

func (m *Manager) Balance() (*big.Float, error) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancelCtx()

	return m.provider.AccountBalance(ctx, m.asset)
}
