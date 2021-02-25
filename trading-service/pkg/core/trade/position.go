package trade

import (
	"github.com/google/uuid"
	"math/big"
	"time"
)

type PositionType int

const (
	LONG PositionType = iota
	SHORT
)

func (pt PositionType) String() string {
	switch pt {
	case LONG:
		return "LONG"
	case SHORT:
		return "SHORT"
	}

	return ""
}

type Position struct {
	ID              uuid.UUID
	Type            PositionType
	EntryPrice      *big.Float
	Size            *big.Float
	TakeProfitPrice *big.Float
	StopLossPrice   *big.Float
	Pair            string
	Exchange        string
	Time            time.Time
}

type Signal struct {
	Type             PositionType
	EntryTarget      *big.Float
	TakeProfitTarget *big.Float
	StopLossTarget   *big.Float
}
