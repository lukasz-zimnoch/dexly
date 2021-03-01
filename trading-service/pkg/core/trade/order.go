package trade

import (
	"github.com/google/uuid"
	"math/big"
	"time"
)

type OrderSide int

const (
	BUY OrderSide = iota
	SELL
)

func (os OrderSide) String() string {
	switch os {
	case BUY:
		return "BUY"
	case SELL:
		return "SELL"
	}

	return ""
}

type Order struct {
	ID       uuid.UUID
	Position *Position
	Side     OrderSide
	Price    *big.Float
	Size     *big.Float
	Time     time.Time
	Executed bool
}