package inmem

import (
	"github.com/google/uuid"
	"github.com/lukasz-zimnoch/dexly/trading"
	"math/big"
)

// TODO: temporary in-mem account repository
type AccountRepository struct{}

func NewAccountRepository() *AccountRepository {
	return &AccountRepository{}
}

func (ar *AccountRepository) Account(id uuid.UUID) (*trading.Account, error) {
	return &trading.Account{
		ID:                 uuid.New(),
		Email:              "test@email.com",
		RiskFactor:         big.NewFloat(0.02),
		OpenPositionsLimit: 1,
	}, nil
}
