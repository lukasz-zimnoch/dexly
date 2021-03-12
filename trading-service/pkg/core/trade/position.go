package trade

import (
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"time"
)

type PositionType int

const (
	LONG PositionType = iota
	SHORT
)

func ParsePositionType(value string) (PositionType, error) {
	switch value {
	case "LONG":
		return LONG, nil
	case "SHORT":
		return SHORT, nil
	}

	return -1, fmt.Errorf("unknown position type: [%v]", value)
}

func (pt PositionType) EntryOrderSide() OrderSide {
	switch pt {
	case LONG:
		return BUY
	case SHORT:
		return SELL
	default:
		panic("unknown position type")
	}
}

func (pt PositionType) ExitOrderSide() OrderSide {
	switch pt {
	case LONG:
		return SELL
	case SHORT:
		return BUY
	default:
		panic("unknown position type")
	}
}

func (pt PositionType) String() string {
	switch pt {
	case LONG:
		return "LONG"
	case SHORT:
		return "SHORT"
	default:
		panic("unknown position type")
	}
}

type PositionStatus int

const (
	OPEN PositionStatus = iota
	CLOSED
)

func ParsePositionStatus(value string) (PositionStatus, error) {
	switch value {
	case "OPEN":
		return OPEN, nil
	case "CLOSED":
		return CLOSED, nil
	}

	return -1, fmt.Errorf("unknown position status: [%v]", value)
}

func (ps PositionStatus) String() string {
	switch ps {
	case OPEN:
		return "OPEN"
	case CLOSED:
		return "CLOSED"
	default:
		panic("unknown position status")
	}
}

type Position struct {
	ID              uuid.UUID
	Type            PositionType
	Status          PositionStatus
	EntryPrice      *big.Float
	Size            *big.Float
	TakeProfitPrice *big.Float
	StopLossPrice   *big.Float
	Pair            string
	Exchange        string
	Time            time.Time
	Orders          []*Order
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
		Status:          OPEN,
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
