package strategy

import (
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/candle"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/order"
	technicalbig "github.com/sdcoffey/big"
	technical "github.com/sdcoffey/techan"
	"math/big"
	"sync"
)

type CandleSource interface {
	Get() []*candle.Candle
}

type Strategy struct {
	recordMutex sync.Mutex
	record      *technical.TradingRecord

	candleSource CandleSource
}

func New(candleSource CandleSource) *Strategy {
	return &Strategy{
		record:       technical.NewTradingRecord(),
		candleSource: candleSource,
	}
}

// TODO: implement proper strategy
func (s Strategy) Propose() (*order.Order, bool) {
	s.recordMutex.Lock()
	defer s.recordMutex.Unlock()

	series := technical.NewTimeSeries()

	for _, currentCandle := range s.candleSource.Get() {
		series.AddCandle(newTechnicalCandle(currentCandle))
	}

	price := technical.NewClosePriceIndicator(series)
	priceEma := technical.NewEMAIndicator(price, 100)

	entryRule := technical.And(
		newNearCrossUpIndicatorRule(priceEma, price),
		technical.PositionNewRule{})

	exitPrice := technical.NewConstantIndicator(1100)

	exitRule := technical.And(
		technical.NewCrossDownIndicatorRule(price, exitPrice),
		technical.PositionOpenRule{})

	rules := technical.RuleStrategy{
		EntryRule: entryRule,
		ExitRule:  exitRule,
	}

	lastIndex := series.LastIndex()
	lastClosePrice := bigFloat(series.LastCandle().ClosePrice)

	if rules.ShouldEnter(lastIndex, s.record) {
		return &order.Order{
			Side:   order.BUY,
			Price:  lastClosePrice,
			Amount: big.NewFloat(100), // TODO: risk evaluation
		}, true
	} else if rules.ShouldExit(lastIndex, s.record) {
		return &order.Order{
			Side:   order.SELL,
			Price:  lastClosePrice,
			Amount: bigFloat(s.record.CurrentPosition().EntranceOrder().Amount),
		}, true
	}

	return nil, false
}

func (s *Strategy) Record(order *order.Order) {
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

func newTechnicalOrder(order *order.Order) technical.Order {
	return technical.Order{
		Side:          technical.OrderSide(order.Side),
		Security:      "",
		Price:         technicalbig.NewFromString(order.Price.String()),
		Amount:        technicalbig.NewFromString(order.Amount.String()),
		ExecutionTime: order.ExecutionTime,
	}
}

func bigFloat(decimal technicalbig.Decimal) *big.Float {
	return big.NewFloat(decimal.Float())
}

func newNearCrossUpIndicatorRule(
	upper, lower technical.Indicator,
) technical.Rule {
	return nearCrossRule{
		upper: upper,
		lower: lower,
		cmp:   1,
	}
}

type nearCrossRule struct {
	upper technical.Indicator
	lower technical.Indicator
	cmp   int
}

func (ncr nearCrossRule) IsSatisfied(
	index int,
	record *technical.TradingRecord,
) bool {
	if index == 0 {
		return false
	}

	current := ncr.lower.Calculate(index).Cmp(ncr.upper.Calculate(index))
	previous := ncr.lower.Calculate(index - 1).Cmp(ncr.upper.Calculate(index - 1))

	if (current == 0 || current == ncr.cmp) &&
		(previous == 0 || previous == -ncr.cmp) {
		return true
	}

	return false
}
