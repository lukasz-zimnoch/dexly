package trading

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"time"
)

type PositionType int

const (
	TypeLong PositionType = iota
	TypeShort
)

func ParsePositionType(value string) (PositionType, error) {
	switch value {
	case "LONG":
		return TypeLong, nil
	case "SHORT":
		return TypeShort, nil
	}

	return -1, fmt.Errorf("unknown position type: [%v]", value)
}

func (pt PositionType) EntryOrderSide() OrderSide {
	switch pt {
	case TypeLong:
		return SideBuy
	case TypeShort:
		return SideSell
	default:
		panic("unknown position type")
	}
}

func (pt PositionType) ExitOrderSide() OrderSide {
	switch pt {
	case TypeLong:
		return SideSell
	case TypeShort:
		return SideBuy
	default:
		panic("unknown position type")
	}
}

func (pt PositionType) String() string {
	switch pt {
	case TypeLong:
		return "LONG"
	case TypeShort:
		return "SHORT"
	default:
		panic("unknown position type")
	}
}

type PositionStatus int

const (
	StatusOpen PositionStatus = iota
	StatusClosed
)

func ParsePositionStatus(value string) (PositionStatus, error) {
	switch value {
	case "OPEN":
		return StatusOpen, nil
	case "CLOSED":
		return StatusClosed, nil
	}

	return -1, fmt.Errorf("unknown position status: [%v]", value)
}

func (ps PositionStatus) String() string {
	switch ps {
	case StatusOpen:
		return "OPEN"
	case StatusClosed:
		return "CLOSED"
	default:
		panic("unknown position status")
	}
}

type PositionFilter struct {
	WorkloadID ID
	Status     PositionStatus
}

type PositionRepository interface {
	CreatePosition(position *Position) error

	UpdatePosition(position *Position) error

	Positions(filter PositionFilter) ([]*Position, error)

	PositionsCount(filter PositionFilter) (int, error)
}

type Position struct {
	ID              ID
	WorkloadID      ID
	Type            PositionType
	Status          PositionStatus
	EntryPrice      *big.Float
	Size            *big.Float
	TakeProfitPrice *big.Float
	StopLossPrice   *big.Float
	Time            time.Time
	Orders          []*Order
}

func (p *Position) OrdersBreakdown() (*Order, *Order, error) {
	ordersCount := len(p.Orders)

	if ordersCount == 0 {
		return nil, nil, nil
	} else if ordersCount == 1 {
		entryOrder := p.Orders[0]

		if entryOrder.Side != p.Type.EntryOrderSide() {
			return nil, nil, fmt.Errorf("entry order has wrong side")
		}

		return entryOrder, nil, nil
	} else if ordersCount == 2 {
		sort.SliceStable(p.Orders, func(i, j int) bool {
			return p.Orders[i].Time.Before(p.Orders[j].Time)
		})

		entryOrder := p.Orders[0]
		exitOrder := p.Orders[1]

		if entryOrder.Side != p.Type.EntryOrderSide() {
			return nil, nil, fmt.Errorf("entry order has wrong side")
		}

		if !entryOrder.Executed {
			return nil, nil, fmt.Errorf(
				"exit order exists despite entry order is not executed yet",
			)
		}

		if exitOrder.Side != p.Type.ExitOrderSide() {
			return nil, nil, fmt.Errorf("exit order has wrong side")
		}

		return entryOrder, exitOrder, nil
	} else {
		return nil, nil, fmt.Errorf("wrong orders count: [%v]", ordersCount)
	}
}

type PositionOpener struct {
	workload           *Workload
	walletItem         *AccountWalletItem
	positionRepository PositionRepository
	idService          IDService
	eventService       EventService
}

func (po *PositionOpener) OpenPosition(
	signal *Signal,
) (*Position, string, error) {
	if signal.Type != TypeLong {
		return nil, "only LONG signals are currently supported", nil
	}

	openPositionsCount, err := po.positionRepository.PositionsCount(
		PositionFilter{
			WorkloadID: po.workload.ID,
			Status:     StatusOpen,
		},
	)
	if err != nil {
		return nil, "", fmt.Errorf(
			"could not count open positions: [%v]",
			err,
		)
	}

	if openPositionsCount >= po.walletItem.OpenPositionsLimit {
		return nil, "open position limit violated", nil
	}

	accountBalance := po.walletItem.Balance
	accountRisk := new(big.Float).Mul(accountBalance, po.walletItem.RiskFactor)
	tradeRisk := new(big.Float).Sub(signal.EntryTarget, signal.StopLossTarget)
	positionSize := new(big.Float).Quo(accountRisk, tradeRisk)

	maxPositionSize := new(big.Float).Quo(accountBalance, signal.EntryTarget)
	if positionSize.Cmp(maxPositionSize) == 1 {
		positionSize = maxPositionSize
	}

	if positionSize.Cmp(big.NewFloat(0)) == 0 {
		return nil, "insufficient funds", nil
	}

	takeProfitPrice := new(big.Float).Mul(
		signal.TakeProfitTarget,
		new(big.Float).Add(big.NewFloat(1), po.walletItem.TakerCommission),
	)

	stopLossPrice := new(big.Float).Mul(
		signal.StopLossTarget,
		new(big.Float).Sub(big.NewFloat(1), po.walletItem.TakerCommission),
	)

	// TODO: Read from exchange info.
	roundToPrecision := func(value *big.Float) *big.Float {
		float, _ := value.Float64()
		precisionPower := math.Pow(10, float64(4))
		return big.NewFloat(math.Round(float*precisionPower) / precisionPower)
	}

	position := &Position{
		ID:              po.idService.NewID(),
		WorkloadID:      po.workload.ID,
		Type:            signal.Type,
		Status:          StatusOpen,
		EntryPrice:      roundToPrecision(signal.EntryTarget),
		Size:            roundToPrecision(positionSize),
		TakeProfitPrice: roundToPrecision(takeProfitPrice),
		StopLossPrice:   roundToPrecision(stopLossPrice),
		Time:            time.Now(),
	}

	err = po.positionRepository.CreatePosition(position)
	if err != nil {
		return nil, "", fmt.Errorf("could not persist position: [%v]", err)
	}

	po.eventService.Publish(NewPositionOpenedEvent(po.workload, position))

	return position, "", nil
}

type PositionCloser struct {
	workload           *Workload
	positionRepository PositionRepository
	eventService       EventService
}

func (pc *PositionCloser) ClosePosition(position *Position) error {
	position.Status = StatusClosed

	if err := pc.positionRepository.UpdatePosition(position); err != nil {
		return fmt.Errorf("could not update position: [%v]", err)
	}

	pc.eventService.Publish(NewPositionClosedEvent(pc.workload, position))

	return nil
}
