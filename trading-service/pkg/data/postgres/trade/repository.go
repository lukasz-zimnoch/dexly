package trade

import (
	"fmt"
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
    	position (id, type, status, entry_price, size, take_profit_price, 
    	          stop_loss_price, pair, exchange, time) 
    	VALUES (:id, :type, :status, :entry_price, :size, :take_profit_price, 
    	        :stop_loss_price, :pair, :exchange, :time)`

	pgPosition, err := toPgPosition(position)
	if err != nil {
		return fmt.Errorf(
			"could not convert position [%v] to pg model: [%v]",
			position.ID,
			err,
		)
	}

	_, err = pr.client.NamedExec(query, pgPosition)
	if err != nil {
		return fmt.Errorf(
			"could not execute command for position [%v]: [%v]",
			position.ID,
			err,
		)
	}

	return nil
}

func (pr *PgRepository) UpdatePosition(position *trade.Position) error {
	query := `UPDATE position SET status = :status WHERE id = :id`

	pgPosition, err := toPgPosition(position)
	if err != nil {
		return fmt.Errorf(
			"could not convert position [%v] to pg model: [%v]",
			position.ID,
			err,
		)
	}

	_, err = pr.client.NamedExec(query, pgPosition)
	if err != nil {
		return fmt.Errorf(
			"could not execute command for position [%v]: [%v]",
			position.ID,
			err,
		)
	}

	return nil
}

func (pr *PgRepository) GetPositions(
	filter trade.PositionFilter,
) ([]*trade.Position, error) {
	var selectResult []struct {
		pgPosition `db:"position"`
		pgOrder    `db:"order"`
	}

	query :=
		`SELECT 
       		p.id "position.id",
       		p.type "position.type",
       		p.status "position.status",
       		p.entry_price "position.entry_price",
       		p.size "position.size",
       		p.take_profit_price "position.take_profit_price",
       		p.stop_loss_price "position.stop_loss_price",
       		p.pair "position.pair",
       		p.exchange "position.exchange",
       		p.time "position.time",
    		o.id "order.id", 
       		o.position_id "order.position_id", 
       		o.side "order.side", 
       		o.price "order.price", 
       		o.size "order.size",
       		o.time "order.time",
       		o.executed "order.executed"
		FROM position p
		LEFT JOIN position_order o ON o.position_id = p.id
		WHERE p.pair = $1 AND p.exchange = $2 AND p.status = $3
		ORDER BY o.time ASC`

	err := pr.client.Select(
		&selectResult,
		query,
		filter.Pair,
		filter.Exchange,
		filter.Status.String(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"could not execute query for filter [%+v]: [%v]",
			filter,
			err,
		)
	}

	positionsByID := make(map[uuid.UUID]*trade.Position)

	for _, result := range selectResult {
		order, err := fromPgOrder(&result.pgOrder)
		if err != nil {
			return nil, fmt.Errorf(
				"could not convert order [%v] from pg model: [%v]",
				result.pgOrder.ID,
				err,
			)
		}

		position, exists := positionsByID[result.pgPosition.ID]
		if !exists {
			position, err = fromPgPosition(&result.pgPosition)
			if err != nil {
				return nil, fmt.Errorf(
					"could not convert position [%v] from pg model: [%v]",
					result.pgPosition.ID,
					err,
				)
			}

			positionsByID[result.pgPosition.ID] = position
		}

		position.Orders = append(position.Orders, order)
	}

	positions := make([]*trade.Position, 0)
	for _, position := range positionsByID {
		positions = append(positions, position)
	}

	return positions, nil
}

func (pr *PgRepository) CountPositions(
	filter trade.PositionFilter,
) (int, error) {
	var count int

	query := `SELECT COUNT(*) FROM position 
		WHERE pair = $1 AND exchange = $2 AND status = $3`

	err := pr.client.Get(
		&count,
		query,
		filter.Pair,
		filter.Exchange,
		filter.Status.String(),
	)
	if err != nil {
		return 0, fmt.Errorf(
			"could not execute query for filter [%+v]: [%v]",
			filter,
			err,
		)
	}

	return count, nil
}

func (pr *PgRepository) CreateOrder(order *trade.Order) error {
	query := `INSERT INTO 
    	position_order (id, position_id, side, price, size, time, executed) 
    	VALUES (:id, :position_id, :side, :price, :size, :time, :executed)`

	pgOrder, err := toPgOrder(order)
	if err != nil {
		return fmt.Errorf(
			"could not convert order [%v] to pg model: [%v]",
			order.ID,
			err,
		)
	}

	_, err = pr.client.NamedExec(query, pgOrder)
	if err != nil {
		return fmt.Errorf(
			"could not execute command for order [%v]: [%v]",
			order.ID,
			err,
		)
	}

	return nil
}

func (pr *PgRepository) UpdateOrder(order *trade.Order) error {
	query := `UPDATE position_order SET executed = :executed WHERE id = :id`

	pgOrder, err := toPgOrder(order)
	if err != nil {
		return fmt.Errorf(
			"could not convert order [%v] to pg model: [%v]",
			order.ID,
			err,
		)
	}

	_, err = pr.client.NamedExec(query, pgOrder)
	if err != nil {
		return fmt.Errorf(
			"could not execute command for order [%v]: [%v]",
			order.ID,
			err,
		)
	}

	return nil
}

type pgPosition struct {
	ID              uuid.UUID
	Type            pgtype.EnumType
	Status          pgtype.EnumType
	EntryPrice      pgtype.Numeric `db:"entry_price"`
	Size            pgtype.Numeric
	TakeProfitPrice pgtype.Numeric `db:"take_profit_price"`
	StopLossPrice   pgtype.Numeric `db:"stop_loss_price"`
	Pair            string
	Exchange        string
	Time            time.Time
}

func toPgPosition(position *trade.Position) (*pgPosition, error) {
	positionType, err := toPgEnum(position.Type.String())
	if err != nil {
		return nil, err
	}

	positionStatus, err := toPgEnum(position.Status.String())
	if err != nil {
		return nil, err
	}

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
		Type:            positionType,
		Status:          positionStatus,
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
	positionTypeString, err := fromPgEnum(pgPosition.Type)
	if err != nil {
		return nil, err
	}

	positionType, err := trade.ParsePositionType(positionTypeString)
	if err != nil {
		return nil, err
	}

	positionStatusString, err := fromPgEnum(pgPosition.Status)
	if err != nil {
		return nil, err
	}

	positionStatus, err := trade.ParsePositionStatus(positionStatusString)
	if err != nil {
		return nil, err
	}

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
		Type:            positionType,
		Status:          positionStatus,
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
	Side       pgtype.EnumType
	Price      pgtype.Numeric
	Size       pgtype.Numeric
	Time       time.Time
	Executed   bool
}

func toPgOrder(order *trade.Order) (*pgOrder, error) {
	orderSide, err := toPgEnum(order.Side.String())
	if err != nil {
		return nil, err
	}

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
		PositionID: order.PositionID,
		Side:       orderSide,
		Price:      price,
		Size:       size,
		Time:       order.Time,
		Executed:   order.Executed,
	}, nil
}

func fromPgOrder(pgOrder *pgOrder) (*trade.Order, error) {
	orderSideString, err := fromPgEnum(pgOrder.Side)
	if err != nil {
		return nil, err
	}

	orderSide, err := trade.ParseOrderSide(orderSideString)
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
		ID:         pgOrder.ID,
		PositionID: pgOrder.PositionID,
		Side:       orderSide,
		Price:      price,
		Size:       size,
		Time:       pgOrder.Time,
		Executed:   pgOrder.Executed,
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

func toPgEnum(value string) (pgtype.EnumType, error) {
	var result pgtype.EnumType

	if err := result.Set(value); err != nil {
		return pgtype.EnumType{}, err
	}

	return result, nil
}

func fromPgEnum(value pgtype.EnumType) (string, error) {
	var result string

	if err := value.AssignTo(&value); err != nil {
		return "", err
	}

	return result, nil
}
