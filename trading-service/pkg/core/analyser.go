package core

import (
	"github.com/sdcoffey/big"
	technical "github.com/sdcoffey/techan"
)

type analyser struct {
	timeSeries        *technical.TimeSeries
	maxTimeSeriesSize int
}

func newAnalyser(maxTimeSeriesSize int) *analyser {
	return &analyser{
		timeSeries:        technical.NewTimeSeries(),
		maxTimeSeriesSize: maxTimeSeriesSize,
	}
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
