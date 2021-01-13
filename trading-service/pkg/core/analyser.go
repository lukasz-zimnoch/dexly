package core

import (
	"github.com/sdcoffey/big"
	technical "github.com/sdcoffey/techan"
)

type analyser struct {
	timeSeries        *technical.TimeSeries
	maxTimeSeriesSize int
	tradingRecord     *technical.TradingRecord
	strategy          technical.Strategy
}

func newAnalyser(maxTimeSeriesSize int) *analyser {
	timeSeries := technical.NewTimeSeries()
	strategy := createStrategy(timeSeries, maxTimeSeriesSize/2)

	return &analyser{
		timeSeries:        timeSeries,
		maxTimeSeriesSize: maxTimeSeriesSize,
		tradingRecord:     technical.NewTradingRecord(),
		strategy:          strategy,
	}
}

// TODO: temporary strategy
func createStrategy(
	timeSeries *technical.TimeSeries,
	unstablePeriod int,
) technical.Strategy {
	indicator := technical.NewClosePriceIndicator(timeSeries)

	entryConstant := technical.NewConstantIndicator(1100)
	exitConstant := technical.NewConstantIndicator(1000)

	entryRule := technical.And(
		technical.NewCrossUpIndicatorRule(entryConstant, indicator),
		technical.PositionNewRule{})

	exitRule := technical.And(
		technical.NewCrossDownIndicatorRule(indicator, exitConstant),
		technical.PositionOpenRule{})

	return technical.RuleStrategy{
		UnstablePeriod: unstablePeriod,
		EntryRule:      entryRule,
		ExitRule:       exitRule,
	}
}

func (a *analyser) checkSignal() (*signal, bool) {
	if shouldEnter := a.strategy.ShouldEnter(
		a.timeSeries.LastIndex(),
		a.tradingRecord,
	); shouldEnter {
		return &signal{}, true
	}

	return nil, false
}

func (a *analyser) addCandles(candles ...*Candle) {
	for _, candle := range candles {
		lastCandle := a.timeSeries.LastCandle()
		newCandle := convertTechnicalCandle(candle)

		if lastCandle != nil && areEqualTechnicalCandles(lastCandle, newCandle) {
			lastCandle.OpenPrice = newCandle.OpenPrice
			lastCandle.ClosePrice = newCandle.ClosePrice
			lastCandle.MaxPrice = newCandle.MaxPrice
			lastCandle.MinPrice = newCandle.MinPrice
			lastCandle.Volume = newCandle.Volume
			lastCandle.TradeCount = newCandle.TradeCount
		} else {
			a.timeSeries.AddCandle(newCandle)

			if len(a.timeSeries.Candles) > a.maxTimeSeriesSize {
				a.deleteCandle(0)
			}
		}
	}
}

func (a *analyser) deleteCandle(index int) bool {
	if index < 0 || index >= len(a.timeSeries.Candles) {
		return false
	}

	copy(a.timeSeries.Candles[index:], a.timeSeries.Candles[index+1:])
	a.timeSeries.Candles[len(a.timeSeries.Candles)-1] = nil
	a.timeSeries.Candles = a.timeSeries.Candles[:len(a.timeSeries.Candles)-1]

	return true
}

func convertTechnicalCandle(candle *Candle) *technical.Candle {
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

func areEqualTechnicalCandles(one, two *technical.Candle) bool {
	return one.Period.Start.Equal(two.Period.Start) &&
		one.Period.End.Equal(two.Period.End)
}
