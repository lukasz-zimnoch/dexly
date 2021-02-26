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
	var selectResult []struct {
		pgPosition `db:"position"`
		pgOrder    `db:"order"`
	}

	query :=
		`SELECT 
    		o.id "order.id", 
       		o.position_id "order.position_id", 
       		o.side "order.side", 
       		o.price "order.price", 
       		o.size "order.size",
       		o.time "order.time",
       		o.executed "order.executed",
       		p.id "position.id",
       		p.type "position.type",
       		p.entry_price "position.entry_price",
       		p.size "position.size",
       		p.take_profit_price "position.take_profit_price",
       		p.stop_loss_price "position.stop_loss_price",
       		p.pair "position.pair",
       		p.exchange "position.exchange",
       		p.time "position.time" 
		FROM position_order o 
		JOIN position p ON p.id = o.position_id 
		WHERE p.pair = $1 AND p.exchange = $2 AND o.executed = $3
		ORDER BY o.time ASC`

	err := pr.client.Select(&selectResult, query, pair, exchange, executed)
	if err != nil {
		return nil, err
	}

	orders := make([]*trade.Order, len(selectResult))
	for i, result := range selectResult {
		orders[i] = fromPgOrder(&result.pgOrder, &result.pgPosition)
	}

	return orders, nil
}

type pgPosition struct {
	ID              uuid.UUID
	Type            int
	EntryPrice      pgtype.Float8 `db:"entry_price"`
	Size            pgtype.Float8
	TakeProfitPrice pgtype.Float8 `db:"take_profit_price"`
	StopLossPrice   pgtype.Float8 `db:"stop_loss_price"`
	Pair            string
	Exchange        string
	Time            time.Time
}

func fromPgPosition(pgPosition *pgPosition) *trade.Position {
	return &trade.Position{
		ID:              pgPosition.ID,
		Type:            trade.PositionType(pgPosition.Type),
		EntryPrice:      fromPgFloat(pgPosition.EntryPrice),
		Size:            fromPgFloat(pgPosition.Size),
		TakeProfitPrice: fromPgFloat(pgPosition.TakeProfitPrice),
		StopLossPrice:   fromPgFloat(pgPosition.StopLossPrice),
		Pair:            pgPosition.Pair,
		Exchange:        pgPosition.Exchange,
		Time:            pgPosition.Time,
	}
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

func fromPgOrder(pgOrder *pgOrder, pgPosition *pgPosition) *trade.Order {
	return &trade.Order{
		ID:       pgOrder.ID,
		Position: fromPgPosition(pgPosition),
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
