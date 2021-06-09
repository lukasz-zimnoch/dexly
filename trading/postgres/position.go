package postgres

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/lukasz-zimnoch/dexly/trading"
	"time"
)

type PositionRepository struct {
	client *Client
}

func NewPositionRepository(client *Client) *PositionRepository {
	return &PositionRepository{client}
}

func (pr *PositionRepository) CreatePosition(position *trading.Position) error {
	query := `INSERT INTO 
    	position (id, type, status, entry_price, size, take_profit_price, 
    	          stop_loss_price, pair, exchange, time) 
    	VALUES (:id, :type, :status, :entry_price, :size, :take_profit_price, 
    	        :stop_loss_price, :pair, :exchange, :time)`

	positionRow, err := new(positionRow).wrap(position)
	if err != nil {
		return fmt.Errorf(
			"could not convert position [%v] to pg row: [%v]",
			position.ID,
			err,
		)
	}

	_, err = pr.client.instance().NamedExec(query, positionRow)
	if err != nil {
		return fmt.Errorf(
			"could not execute command for position [%v]: [%v]",
			position.ID,
			err,
		)
	}

	return nil
}

func (pr *PositionRepository) UpdatePosition(position *trading.Position) error {
	query := `UPDATE position SET status = :status WHERE id = :id`

	positionRow, err := new(positionRow).wrap(position)
	if err != nil {
		return fmt.Errorf(
			"could not convert position [%v] to pg row: [%v]",
			position.ID,
			err,
		)
	}

	_, err = pr.client.instance().NamedExec(query, positionRow)
	if err != nil {
		return fmt.Errorf(
			"could not execute command for position [%v]: [%v]",
			position.ID,
			err,
		)
	}

	return nil
}

func (pr *PositionRepository) Positions(
	filter trading.PositionFilter,
) ([]*trading.Position, error) {
	var selectResult []struct {
		positionRow `db:"position"`
		orderRow    `db:"order"`
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

	err := pr.client.instance().Select(
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

	positionsByID := make(map[uuid.UUID]*trading.Position)

	for _, result := range selectResult {
		order, err := result.orderRow.unwrap()
		if err != nil {
			return nil, fmt.Errorf(
				"could not convert order [%v] from pg row: [%v]",
				result.orderRow.ID,
				err,
			)
		}

		position, exists := positionsByID[result.positionRow.ID]
		if !exists {
			position, err = result.positionRow.unwrap()
			if err != nil {
				return nil, fmt.Errorf(
					"could not convert position [%v] from pg row: [%v]",
					result.positionRow.ID,
					err,
				)
			}

			positionsByID[result.positionRow.ID] = position
		}

		order.Position = position
		position.Orders = append(position.Orders, order)
	}

	positions := make([]*trading.Position, 0)
	for _, position := range positionsByID {
		positions = append(positions, position)
	}

	return positions, nil
}

func (pr *PositionRepository) PositionsCount(
	filter trading.PositionFilter,
) (int, error) {
	var count int

	query := `SELECT COUNT(*) FROM position 
		WHERE pair = $1 AND exchange = $2 AND status = $3`

	err := pr.client.instance().Get(
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

type positionRow struct {
	ID              uuid.UUID
	Type            string
	Status          string
	EntryPrice      pgtype.Numeric `db:"entry_price"`
	Size            pgtype.Numeric
	TakeProfitPrice pgtype.Numeric `db:"take_profit_price"`
	StopLossPrice   pgtype.Numeric `db:"stop_loss_price"`
	Pair            string
	Exchange        string
	Time            time.Time
}

func (pr *positionRow) wrap(
	position *trading.Position,
) (*positionRow, error) {
	entryPrice, err := floatToNumeric(position.EntryPrice)
	if err != nil {
		return nil, err
	}

	size, err := floatToNumeric(position.Size)
	if err != nil {
		return nil, err
	}

	takeProfitPrice, err := floatToNumeric(position.TakeProfitPrice)
	if err != nil {
		return nil, err
	}

	stopLossPrice, err := floatToNumeric(position.StopLossPrice)
	if err != nil {
		return nil, err
	}

	pr.ID = position.ID
	pr.Type = position.Type.String()
	pr.Status = position.Status.String()
	pr.EntryPrice = entryPrice
	pr.Size = size
	pr.TakeProfitPrice = takeProfitPrice
	pr.StopLossPrice = stopLossPrice
	pr.Pair = position.Pair
	pr.Exchange = position.Exchange
	pr.Time = position.Time

	return pr, nil
}

func (pr *positionRow) unwrap() (*trading.Position, error) {
	positionType, err := trading.ParsePositionType(pr.Type)
	if err != nil {
		return nil, err
	}

	positionStatus, err := trading.ParsePositionStatus(pr.Status)
	if err != nil {
		return nil, err
	}

	entryPrice, err := numericToFloat(pr.EntryPrice)
	if err != nil {
		return nil, err
	}

	size, err := numericToFloat(pr.Size)
	if err != nil {
		return nil, err
	}

	takeProfitPrice, err := numericToFloat(pr.TakeProfitPrice)
	if err != nil {
		return nil, err
	}

	stopLossPrice, err := numericToFloat(pr.StopLossPrice)
	if err != nil {
		return nil, err
	}

	return &trading.Position{
		ID:              pr.ID,
		Type:            positionType,
		Status:          positionStatus,
		EntryPrice:      entryPrice,
		Size:            size,
		TakeProfitPrice: takeProfitPrice,
		StopLossPrice:   stopLossPrice,
		Pair:            pr.Pair,
		Exchange:        pr.Exchange,
		Time:            pr.Time,
	}, nil
}