package postgres

import (
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/lukasz-zimnoch/dexly/trading"
	"time"
)

type PositionRepository struct {
	client    *Client
	idService trading.IDService
}

func NewPositionRepository(
	client *Client,
	idService trading.IDService,
) *PositionRepository {
	return &PositionRepository{client, idService}
}

func (pr *PositionRepository) CreatePosition(position *trading.Position) error {
	query := `INSERT INTO 
    	position (id, workload_id, type, status, entry_price, size,  
    	          take_profit_price, stop_loss_price, time) 
    	VALUES (:id, :workload_id, :type, :status, :entry_price, :size,  
    	        :take_profit_price, :stop_loss_price, :time)`

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
       		p.workload_id "position.workload_id",
       		p.type "position.type",
       		p.status "position.status",
       		p.entry_price "position.entry_price",
       		p.size "position.size",
       		p.take_profit_price "position.take_profit_price",
       		p.stop_loss_price "position.stop_loss_price",
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
		WHERE p.workload_id = $1 AND p.status = $2
		ORDER BY o.time ASC`

	err := pr.client.instance().Select(
		&selectResult,
		query,
		filter.WorkloadID,
		filter.Status.String(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"could not execute query for filter [%+v]: [%v]",
			filter,
			err,
		)
	}

	positionsByID := make(map[string]*trading.Position)

	for _, result := range selectResult {
		order, err := result.orderRow.unwrap(pr.idService)
		if err != nil {
			return nil, fmt.Errorf(
				"could not convert order [%v] from pg row: [%v]",
				result.orderRow.ID,
				err,
			)
		}

		position, exists := positionsByID[result.positionRow.ID]
		if !exists {
			position, err = result.positionRow.unwrap(pr.idService)
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
		WHERE workload_id = $1 AND status = $2`

	err := pr.client.instance().Get(
		&count,
		query,
		filter.WorkloadID,
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
	ID              string
	WorkloadID      string `db:"workload_id"`
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

	pr.ID = position.ID.String()
	pr.WorkloadID = position.WorkloadID.String()
	pr.Type = position.Type.String()
	pr.Status = position.Status.String()
	pr.EntryPrice = entryPrice
	pr.Size = size
	pr.TakeProfitPrice = takeProfitPrice
	pr.StopLossPrice = stopLossPrice
	pr.Time = position.Time

	return pr, nil
}

func (pr *positionRow) unwrap(
	idService trading.IDService,
) (*trading.Position, error) {
	ID, err := idService.NewIDFromString(pr.ID)
	if err != nil {
		return nil, err
	}

	workloadID, err := idService.NewIDFromString(pr.WorkloadID)
	if err != nil {
		return nil, err
	}

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
		ID:              ID,
		WorkloadID:      workloadID,
		Type:            positionType,
		Status:          positionStatus,
		EntryPrice:      entryPrice,
		Size:            size,
		TakeProfitPrice: takeProfitPrice,
		StopLossPrice:   stopLossPrice,
		Time:            pr.Time,
	}, nil
}
