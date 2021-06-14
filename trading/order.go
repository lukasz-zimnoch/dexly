package trading

import (
	"fmt"
	"math/big"
	"time"
)

type OrderSide int

const (
	SideBuy OrderSide = iota
	SideSell
)

func ParseOrderSide(value string) (OrderSide, error) {
	switch value {
	case "BUY":
		return SideBuy, nil
	case "SELL":
		return SideSell, nil
	}

	return -1, fmt.Errorf("unknown order side: [%v]", value)
}

func (os OrderSide) String() string {
	switch os {
	case SideBuy:
		return "BUY"
	case SideSell:
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
	ID       ID
	Position *Position
	Side     OrderSide
	Price    *big.Float
	Size     *big.Float
	Time     time.Time
	Executed bool
}

type OrderFactory struct {
	orderRepository OrderRepository
	idService       IDService
}

func (of *OrderFactory) CreateEntryOrder(
	position *Position,
) (*Order, error) {
	order := &Order{
		ID:       of.idService.NewID(),
		Position: position,
		Side:     position.Type.EntryOrderSide(),
		Price:    position.EntryPrice,
		Size:     position.Size,
		Time:     time.Now(),
		Executed: false,
	}

	if err := of.orderRepository.CreateOrder(order); err != nil {
		return nil, fmt.Errorf("could not persist order: [%v]", err)
	}

	return order, nil
}

func (of *OrderFactory) CreateExitOrder(
	position *Position,
	price *big.Float,
) (*Order, error) {
	order := &Order{
		ID:       of.idService.NewID(),
		Position: position,
		Side:     position.Type.ExitOrderSide(),
		Price:    price,
		Size:     position.Size,
		Time:     time.Now(),
		Executed: false,
	}

	if err := of.orderRepository.CreateOrder(order); err != nil {
		return nil, fmt.Errorf("could not persist order: [%v]", err)
	}

	return order, nil
}

type OrderExecutionRecorder struct {
	orderRepository OrderRepository
}

func (oer *OrderExecutionRecorder) recordOrderExecution(order *Order) error {
	order.Executed = true

	if err := oer.orderRepository.UpdateOrder(order); err != nil {
		return fmt.Errorf("could not update order: [%v]", err)
	}

	return nil
}
