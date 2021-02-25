package trade

import (
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/trade"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/data/postgres"
	"math/big"
	"time"
)

type PgRepository struct {
	client *postgres.Client
}

func NewPgRepository(client *postgres.Client) *PgRepository {
	return &PgRepository{client}
}

func (pr *PgRepository) CreatePosition(position *trade.Position) error {
	// TODO: implementation
	return nil
}

func (pr *PgRepository) CreateOrder(order *trade.Order) error {
	// TODO: implementation
	query := ""

	_, err := pr.client.NamedExec(query, toPgOrder(order))

	return err
}

func (pr *PgRepository) UpdateOrder(order *trade.Order) error {
	// TODO: implementation
	return nil
}

func (pr *PgRepository) GetOrders(
	pair string,
	exchange string,
	executed bool,
) ([]*trade.Order, error) {
	// TODO: implementation
	query := ""
	pgOrders := make([]pgOrder, 0)

	err := pr.client.Select(pgOrders, query, pair, exchange, executed)
	if err != nil {
		return nil, err
	}

	orders := make([]*trade.Order, len(pgOrders))
	for i, pgOrder := range pgOrders {
		orders[i] = fromPgOrder(&pgOrder)
	}

	return orders, nil
}

type pgOrder struct {
	ID         uuid.UUID
	PositionID uuid.UUID `db:"position_id"`
	Side       int
	Price      pgtype.Float8
	Size       pgtype.Float8
	Time       time.Time
	Executed   bool
}

func toPgOrder(order *trade.Order) *pgOrder {
	return &pgOrder{
		ID:         order.ID,
		PositionID: order.Position.ID,
		Side:       int(order.Side),
		Price:      toPgFloat(order.Price),
		Size:       toPgFloat(order.Size),
		Time:       order.Time,
		Executed:   order.Executed,
	}
}

func fromPgOrder(pgOrder *pgOrder) *trade.Order {
	return &trade.Order{
		ID:       pgOrder.ID,
		Position: nil, // TODO: fetch position
		Side:     trade.OrderSide(pgOrder.Side),
		Price:    fromPgFloat(pgOrder.Price),
		Size:     fromPgFloat(pgOrder.Size),
		Time:     pgOrder.Time,
		Executed: pgOrder.Executed,
	}
}

func toPgFloat(value *big.Float) pgtype.Float8 {
	valueFloat, _ := value.Float64()

	return pgtype.Float8{
		Float:  valueFloat,
		Status: pgtype.Present,
	}
}

func fromPgFloat(value pgtype.Float8) *big.Float {
	if valueFloat, ok := value.Get().(float64); ok {
		return big.NewFloat(valueFloat)
	}

	return nil
}
