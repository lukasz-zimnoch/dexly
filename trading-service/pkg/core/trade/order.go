package trade

import (
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"time"
)

type OrderSide int

const (
	BUY OrderSide = iota
	SELL
)

func ParseOrderSide(value string) (OrderSide, error) {
	switch value {
	case "BUY":
		return BUY, nil
	case "SELL":
		return SELL, nil
	}

	return -1, fmt.Errorf("unknown order side: [%v]", value)
}

func (os OrderSide) String() string {
	switch os {
	case BUY:
		return "BUY"
	case SELL:
		return "SELL"
	default:
		panic("unknown order side")
	}
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

func NewOrder(
	position *Position,
	orderSide OrderSide,
	price *big.Float,
	size *big.Float,
) *Order {
	return &Order{
		ID:       uuid.New(),
		Position: position,
		Side:     orderSide,
		Price:    price,
		Size:     size,
		Time:     time.Now(),
		Executed: false,
	}
}
