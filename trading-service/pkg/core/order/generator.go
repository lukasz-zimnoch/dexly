package order

import (
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/candle"
	technicalbig "github.com/sdcoffey/big"
	technical "github.com/sdcoffey/techan"
	"math/big"
	"sync"
	"time"
)

type Side int

const (
	BUY Side = iota
	SELL
)

func (s Side) String() string {
	switch s {
	case BUY:
		return "BUY"
	case SELL:
		return "SELL"
	}

	return ""
}

type Order struct {
	Side         Side
	Price        *big.Float
	Amount       *big.Float
	CreationTime time.Time
}

func newBuyOrder(price, amount *big.Float) *Order {
	return &Order{
		Side:         BUY,
		Price:        price,
		Amount:       amount,
		CreationTime: time.Now(),
	}
}

func newSellOrder(price, amount *big.Float) *Order {
	return &Order{
		Side:         SELL,
		Price:        price,
		Amount:       amount,
		CreationTime: time.Now(),
	}
}

func (o *Order) String() string {
	return fmt.Sprintf(
		"side: %v, amount: %v, price: %v",
		o.Side,
		o.Amount.String(),
		o.Price.String(),
	)
}

type candleSource interface {
	Candles() []*candle.Candle
}

type Generator struct {
	recordMutex sync.Mutex
	record      *technical.TradingRecord

	candleSource candleSource
}

func NewGenerator(candleSource candleSource) *Generator {
	return &Generator{
		record:       technical.NewTradingRecord(),
		candleSource: candleSource,
	}
}

func (s Generator) Generate() (*Order, bool) {
	s.recordMutex.Lock()
	defer s.recordMutex.Unlock()

	series := technical.NewTimeSeries()

	for _, currentCandle := range s.candleSource.Candles() {
		series.AddCandle(newTechnicalCandle(currentCandle))
	}

	strategy := evaluateStrategy(series)

	lastIndex := series.LastIndex()
	lastPrice := big.NewFloat(series.LastCandle().ClosePrice.Float())

	if strategy.ShouldEnter(lastIndex, s.record) {
		amount := big.NewFloat(100) // TODO: risk evaluation
		return newBuyOrder(lastPrice, amount), true
	} else if strategy.ShouldExit(lastIndex, s.record) {
		entranceOrder := s.record.CurrentPosition().EntranceOrder()
		amount := big.NewFloat(entranceOrder.Amount.Float())
		return newSellOrder(lastPrice, amount), true
	}

	return nil, false
}

func (s *Generator) RecordExecution(order *Order) {
	s.recordMutex.Lock()
	defer s.recordMutex.Unlock()

	s.record.Operate(newTechnicalOrder(order))
}

func newTechnicalCandle(candle *candle.Candle) *technical.Candle {
	period := technical.TimePeriod{
		Start: candle.OpenTime,
		End:   candle.CloseTime,
	}

	technicalCandle := technical.NewCandle(period)

	technicalCandle.OpenPrice = technicalbig.NewFromString(candle.OpenPrice)
	technicalCandle.ClosePrice = technicalbig.NewFromString(candle.ClosePrice)
	technicalCandle.MaxPrice = technicalbig.NewFromString(candle.MaxPrice)
	technicalCandle.MinPrice = technicalbig.NewFromString(candle.MinPrice)
	technicalCandle.Volume = technicalbig.NewFromString(candle.Volume)
	technicalCandle.TradeCount = candle.TradeCount

	return technicalCandle
}

func newTechnicalOrder(order *Order) technical.Order {
	return technical.Order{
		Side:          technical.OrderSide(order.Side),
		Security:      "",
		Price:         technicalbig.NewFromString(order.Price.String()),
		Amount:        technicalbig.NewFromString(order.Amount.String()),
		ExecutionTime: order.CreationTime,
	}
}
