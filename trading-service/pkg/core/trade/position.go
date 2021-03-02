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

func NewPosition(
	positionType PositionType,
	entryPrice *big.Float,
	size *big.Float,
	takeProfitPrice *big.Float,
	stopLossPrice *big.Float,
	pair string,
	exchange string,
) *Position {
	return &Position{
		ID:              uuid.New(),
		Type:            positionType,
		EntryPrice:      entryPrice,
		Size:            size,
		TakeProfitPrice: takeProfitPrice,
		StopLossPrice:   stopLossPrice,
		Pair:            pair,
		Exchange:        exchange,
		Time:            time.Now(),
	}
}

type Signal struct {
	Type             PositionType
	EntryTarget      *big.Float
	TakeProfitTarget *big.Float
	StopLossTarget   *big.Float
}
