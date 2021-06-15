package postgres

import (
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/lukasz-zimnoch/dexly/trading"
	"time"
)

type OrderRepository struct {
	client    *Client
	idService trading.IDService
}

func NewOrderRepository(
	client *Client,
	idService trading.IDService,
) *OrderRepository {
	return &OrderRepository{client, idService}
}

func (or *OrderRepository) CreateOrder(order *trading.Order) error {
	query := `INSERT INTO 
    	position_order (id, position_id, side, price, size, time, executed) 
    	VALUES (:id, :position_id, :side, :price, :size, :time, :executed)`

	orderRow, err := new(orderRow).wrap(order)
	if err != nil {
		return fmt.Errorf(
			"could not convert order [%v] to pg row: [%v]",
			order.ID,
			err,
		)
	}

	_, err = or.client.instance().NamedExec(query, orderRow)
	if err != nil {
		return fmt.Errorf(
			"could not execute command for order [%v]: [%v]",
			order.ID,
			err,
		)
	}

	return nil
}

func (or *OrderRepository) UpdateOrder(order *trading.Order) error {
	query := `UPDATE position_order SET executed = :executed WHERE id = :id`

	orderRow, err := new(orderRow).wrap(order)
	if err != nil {
		return fmt.Errorf(
			"could not convert order [%v] to pg row: [%v]",
			order.ID,
			err,
		)
	}

	_, err = or.client.instance().NamedExec(query, orderRow)
	if err != nil {
		return fmt.Errorf(
			"could not execute command for order [%v]: [%v]",
			order.ID,
			err,
		)
	}

	return nil
}

type orderRow struct {
	ID         string
	PositionID string `db:"position_id"`
	Side       string
	Price      pgtype.Numeric
	Size       pgtype.Numeric
	Time       time.Time
	Executed   bool
}

func (or *orderRow) wrap(order *trading.Order) (*orderRow, error) {
	price, err := floatToNumeric(order.Price)
	if err != nil {
		return nil, err
	}

	size, err := floatToNumeric(order.Size)
	if err != nil {
		return nil, err
	}

	or.ID = order.ID.String()
	or.PositionID = order.Position.ID.String()
	or.Side = order.Side.String()
	or.Price = price
	or.Size = size
	or.Time = order.Time
	or.Executed = order.Executed

	return or, nil
}

func (or *orderRow) unwrap(
	idService trading.IDService,
) (*trading.Order, error) {
	ID, err := idService.NewIDFromString(or.ID)
	if err != nil {
		return nil, err
	}

	orderSide, err := trading.ParseOrderSide(or.Side)
	if err != nil {
		return nil, err
	}

	price, err := numericToFloat(or.Price)
	if err != nil {
		return nil, err
	}

	size, err := numericToFloat(or.Size)
	if err != nil {
		return nil, err
	}

	return &trading.Order{
		ID:       ID,
		Position: nil, // Position should be set outside.
		Side:     orderSide,
		Price:    price,
		Size:     size,
		Time:     or.Time,
		Executed: or.Executed,
	}, nil
}
