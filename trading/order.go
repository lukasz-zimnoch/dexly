package trading

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

type OrderRepository interface {
	CreateOrder(order *Order) error

	UpdateOrder(order *Order) error
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

type OrderFactory struct {
	repository OrderRepository
}

func (of *OrderFactory) CreateEntryOrder(
	position *Position,
) (*Order, error) {
	order := &Order{
		ID:       uuid.New(),
		Position: position,
		Side:     position.Type.EntryOrderSide(),
		Price:    position.EntryPrice,
		Size:     position.Size,
		Time:     time.Now(),
		Executed: false,
	}

	if err := of.repository.CreateOrder(order); err != nil {
		return nil, fmt.Errorf("could not persist order: [%v]", err)
	}

	return order, nil
}

func (of *OrderFactory) CreateExitOrder(
	position *Position,
	price *big.Float,
) (*Order, error) {
	order := &Order{
		ID:       uuid.New(),
		Position: position,
		Side:     position.Type.ExitOrderSide(),
		Price:    price,
		Size:     position.Size,
		Time:     time.Now(),
		Executed: false,
	}

	if err := of.repository.CreateOrder(order); err != nil {
		return nil, fmt.Errorf("could not persist order: [%v]", err)
	}

	return order, nil
}

type OrderExecutionNoter struct {
	repository OrderRepository
}

func (oen *OrderExecutionNoter) NoteOrderExecution(order *Order) error {
	order.Executed = true

	if err := oen.repository.UpdateOrder(order); err != nil {
		return fmt.Errorf("could not update order: [%v]", err)
	}

	return nil
}
