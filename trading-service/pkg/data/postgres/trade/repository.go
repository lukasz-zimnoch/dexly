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
	query := `INSERT INTO 
    	position (id, type, entry_price, size, take_profit_price, 
    	          stop_loss_price, pair, exchange, time) 
    	VALUES (:id, :type, :entry_price, :size, :take_profit_price, 
    	        :stop_loss_price, :pair, :exchange, :time)`

	pgPosition, err := toPgPosition(position)
	if err != nil {
		return err
	}

	_, err = pr.client.NamedExec(query, pgPosition)

	return err
}

func (pr *PgRepository) CreateOrder(order *trade.Order) error {
	query := `INSERT INTO 
    	position_order (id, position_id, side, price, size, time, executed) 
    	VALUES (:id, :position_id, :side, :price, :size, :time, :executed)`

	pgOrder, err := toPgOrder(order)
	if err != nil {
		return err
	}

	_, err = pr.client.NamedExec(query, pgOrder)

	return err
}

func (pr *PgRepository) UpdateOrder(order *trade.Order) error {
	query := `UPDATE position_order SET executed = :executed WHERE id = :id`

	pgOrder, err := toPgOrder(order)
	if err != nil {
		return err
	}

	_, err = pr.client.NamedExec(query, pgOrder)

	return err
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
		order, err := fromPgOrder(&result.pgOrder, &result.pgPosition)
		if err != nil {
			return nil, err
		}

		orders[i] = order
	}

	return orders, nil
}

type pgPosition struct {
	ID              uuid.UUID
	Type            int
	EntryPrice      pgtype.Numeric `db:"entry_price"`
	Size            pgtype.Numeric
	TakeProfitPrice pgtype.Numeric `db:"take_profit_price"`
	StopLossPrice   pgtype.Numeric `db:"stop_loss_price"`
	Pair            string
	Exchange        string
	Time            time.Time
}

func toPgPosition(position *trade.Position) (*pgPosition, error) {
	entryPrice, err := toPgNumeric(position.EntryPrice)
	if err != nil {
		return nil, err
	}

	size, err := toPgNumeric(position.Size)
	if err != nil {
		return nil, err
	}

	takeProfitPrice, err := toPgNumeric(position.TakeProfitPrice)
	if err != nil {
		return nil, err
	}

	stopLossPrice, err := toPgNumeric(position.StopLossPrice)
	if err != nil {
		return nil, err
	}

	return &pgPosition{
		ID:              position.ID,
		Type:            int(position.Type),
		EntryPrice:      entryPrice,
		Size:            size,
		TakeProfitPrice: takeProfitPrice,
		StopLossPrice:   stopLossPrice,
		Pair:            position.Pair,
		Exchange:        position.Exchange,
		Time:            position.Time,
	}, nil
}

func fromPgPosition(pgPosition *pgPosition) (*trade.Position, error) {
	entryPrice, err := fromPgNumeric(pgPosition.EntryPrice)
	if err != nil {
		return nil, err
	}

	size, err := fromPgNumeric(pgPosition.Size)
	if err != nil {
		return nil, err
	}

	takeProfitPrice, err := fromPgNumeric(pgPosition.TakeProfitPrice)
	if err != nil {
		return nil, err
	}

	stopLossPrice, err := fromPgNumeric(pgPosition.StopLossPrice)
	if err != nil {
		return nil, err
	}

	return &trade.Position{
		ID:              pgPosition.ID,
		Type:            trade.PositionType(pgPosition.Type),
		EntryPrice:      entryPrice,
		Size:            size,
		TakeProfitPrice: takeProfitPrice,
		StopLossPrice:   stopLossPrice,
		Pair:            pgPosition.Pair,
		Exchange:        pgPosition.Exchange,
		Time:            pgPosition.Time,
	}, nil
}

type pgOrder struct {
	ID         uuid.UUID
	PositionID uuid.UUID `db:"position_id"`
	Side       int
	Price      pgtype.Numeric
	Size       pgtype.Numeric
	Time       time.Time
	Executed   bool
}

func toPgOrder(order *trade.Order) (*pgOrder, error) {
	price, err := toPgNumeric(order.Price)
	if err != nil {
		return nil, err
	}

	size, err := toPgNumeric(order.Size)
	if err != nil {
		return nil, err
	}

	return &pgOrder{
		ID:         order.ID,
		PositionID: order.Position.ID,
		Side:       int(order.Side),
		Price:      price,
		Size:       size,
		Time:       order.Time,
		Executed:   order.Executed,
	}, nil
}

func fromPgOrder(
	pgOrder *pgOrder,
	pgPosition *pgPosition,
) (*trade.Order, error) {
	position, err := fromPgPosition(pgPosition)
	if err != nil {
		return nil, err
	}

	price, err := fromPgNumeric(pgOrder.Price)
	if err != nil {
		return nil, err
	}

	size, err := fromPgNumeric(pgOrder.Size)
	if err != nil {
		return nil, err
	}

	return &trade.Order{
		ID:       pgOrder.ID,
		Position: position,
		Side:     trade.OrderSide(pgOrder.Side),
		Price:    price,
		Size:     size,
		Time:     pgOrder.Time,
		Executed: pgOrder.Executed,
	}, nil
}

func toPgNumeric(value *big.Float) (pgtype.Numeric, error) {
	var result pgtype.Numeric
	valueFloat, _ := value.Float64()

	if err := result.Set(valueFloat); err != nil {
		return pgtype.Numeric{}, err
	}

	return result, nil
}

func fromPgNumeric(value pgtype.Numeric) (*big.Float, error) {
	var result float64

	if err := value.AssignTo(&result); err != nil {
		return nil, err
	}

	return big.NewFloat(result), nil
}