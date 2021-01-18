package core

import (
	"github.com/sdcoffey/big"
	technical "github.com/sdcoffey/techan"
)

type ordersRegistry struct {
	delegate *technical.TradingRecord
}

func newOrdersRegistry() *ordersRegistry {
	return &ordersRegistry{technical.NewTradingRecord()}
}

type strategy struct {
	index    int
	delegate technical.Strategy
}

func (s strategy) shouldEnter(registry *ordersRegistry) bool {
	return s.delegate.ShouldEnter(s.index, registry.delegate)
}

func (s strategy) shouldExit(registry *ordersRegistry) bool {
	return s.delegate.ShouldExit(s.index, registry.delegate)
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
		index: series.LastIndex(),
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

	technicalCandle.OpenPrice = big.NewFromString(candle.OpenPrice)
	technicalCandle.ClosePrice = big.NewFromString(candle.ClosePrice)
	technicalCandle.MaxPrice = big.NewFromString(candle.MaxPrice)
	technicalCandle.MinPrice = big.NewFromString(candle.MinPrice)
	technicalCandle.Volume = big.NewFromString(candle.Volume)
	technicalCandle.TradeCount = candle.TradeCount

	return technicalCandle
}
