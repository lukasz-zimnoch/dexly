package account

import (
	"context"
	"math/big"
	"time"
)

type Supplier interface {
	AccountBalance(
		ctx context.Context,
		asset string,
	) (*big.Float, error)
}

type Manager struct {
	supplier   Supplier
	asset      string
	riskFactor *big.Float
}

func NewManager(supplier Supplier, asset string) *Manager {
	return &Manager{
		supplier:   supplier,
		asset:      asset,
		riskFactor: big.NewFloat(0.02),
	}
}

func (m *Manager) Balance() (*big.Float, error) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancelCtx()

	return m.supplier.AccountBalance(ctx, m.asset)
}

func (m *Manager) RiskFactor() *big.Float {
	return m.riskFactor
}
