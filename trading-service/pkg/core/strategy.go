package core

import (
	technicalbig "github.com/sdcoffey/big"
	technical "github.com/sdcoffey/techan"
	"math/big"
)

type signal struct {
	side   orderSide
	price  *big.Float
	amount *big.Float
}

type strategy struct {
	series   *technical.TimeSeries
	delegate technical.Strategy
}

func (s strategy) run(registry *ordersRegistry) (*signal, bool) {
	currentPosition := registry.delegate.CurrentPosition()
	lastIndex := s.series.LastIndex()
	lastCandle := s.series.LastCandle()

	if currentPosition.IsNew() {
		if s.delegate.ShouldEnter(lastIndex, registry.delegate) {
			return &signal{
				side:   BUY,
				price:  asFloat(lastCandle.ClosePrice),
				amount: big.NewFloat(100), // TODO: risk evaluation
			}, true
		}
	} else if currentPosition.IsOpen() {
		if s.delegate.ShouldExit(lastIndex, registry.delegate) {
			return &signal{
				side:   SELL,
				price:  asFloat(lastCandle.ClosePrice),
				amount: asFloat(currentPosition.EntranceOrder().Amount),
			}, true
		}
	}

	return nil, false
}

func evaluateStrategy(candles []*Candle) *strategy {
	series := technical.NewTimeSeries()

	for _, candle := range candles {
		series.AddCandle(newTechnicalCandle(candle))
	}

	indicator := technical.NewClosePriceIndicator(series)

	entryConstant := technical.NewConstantIndicator(1300)
	exitConstant := technical.NewConstantIndicator(1100)

	entryRule := technical.And(
		technical.NewCrossUpIndicatorRule(entryConstant, indicator),
		technical.PositionNewRule{})

	exitRule := technical.And(
		technical.NewCrossDownIndicatorRule(indicator, exitConstant),
		technical.PositionOpenRule{})

	return &strategy{
		series: series,
		delegate: technical.RuleStrategy{
			EntryRule: entryRule,
			ExitRule:  exitRule,
		},
	}
}

func newTechnicalCandle(candle *Candle) *technical.Candle {
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

func asFloat(decimal technicalbig.Decimal) *big.Float {
	return big.NewFloat(decimal.Float())
}
