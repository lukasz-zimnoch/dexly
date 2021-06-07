package trading

import (
	"fmt"
	"github.com/google/uuid"
	"math"
	"math/big"
	"sort"
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

type PositionFilter struct {
	Pair     string
	Exchange string
	Status   PositionStatus
}

type PositionRepository interface {
	CreatePosition(position *Position) error

	UpdatePosition(position *Position) error

	Positions(filter PositionFilter) ([]*Position, error)

	PositionsCount(filter PositionFilter) (int, error)
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
	repository PositionRepository
}

func (po *PositionOpener) OpenPosition(
	signal *Signal,
	account *ExchangeAccount,
) (*Position, string, error) {
	if signal.Type != LONG {
		return nil, "only LONG signals are currently supported", nil
	}

	openPositionsCount, err := po.repository.PositionsCount(
		PositionFilter{
			Pair:     signal.Pair.String(),
			Exchange: account.Exchange,
			Status:   OPEN,
		},
	)
	if err != nil {
		return nil, "", fmt.Errorf(
			"could not count open positions: [%v]",
			err,
		)
	}

	if openPositionsCount >= account.OpenPositionsLimit {
		return nil, "open position limit violated", nil
	}

	accountBalance := account.AssetBalance(signal.Pair.Quote)
	accountRisk := new(big.Float).Mul(accountBalance, account.RiskFactor)
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
		new(big.Float).Add(big.NewFloat(1), account.TakerCommission),
	)

	stopLossPrice := new(big.Float).Mul(
		signal.StopLossTarget,
		new(big.Float).Sub(big.NewFloat(1), account.TakerCommission),
	)

	// TODO: read from exchange info
	roundToPrecision := func(value *big.Float) *big.Float {
		float, _ := value.Float64()
		precisionPower := math.Pow(10, float64(4))
		return big.NewFloat(math.Round(float*precisionPower) / precisionPower)
	}

	position := &Position{
		ID:              uuid.New(),
		Type:            signal.Type,
		Status:          OPEN,
		EntryPrice:      roundToPrecision(signal.EntryTarget),
		Size:            roundToPrecision(positionSize),
		TakeProfitPrice: roundToPrecision(takeProfitPrice),
		StopLossPrice:   roundToPrecision(stopLossPrice),
		Pair:            signal.Pair.String(),
		Exchange:        account.Exchange,
		Time:            time.Now(),
	}

	err = po.repository.CreatePosition(position)
	if err != nil {
		return nil, "", fmt.Errorf("could not persist position: [%v]", err)
	}

	return position, "", nil
}

type PositionCloser struct {
	repository PositionRepository
}

func (pc *PositionCloser) ClosePosition(position *Position) error {
	position.Status = CLOSED

	if err := pc.repository.UpdatePosition(position); err != nil {
		return fmt.Errorf("could not update position: [%v]", err)
	}

	return nil
}
