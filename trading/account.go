package trading

import (
	"github.com/google/uuid"
	"math/big"
)

type Account struct {
	ID                 uuid.UUID
	Email              string
	RiskFactor         *big.Float
	OpenPositionsLimit int
}

type AccountRepository interface {
	Account(id uuid.UUID) (*Account, error)
}
