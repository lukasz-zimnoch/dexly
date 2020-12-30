package core

import (
	"fmt"
	"github.com/sdcoffey/big"
	technical "github.com/sdcoffey/techan"
)

type analyser struct {
	timeSeries *technical.TimeSeries
}

func newAnalyser() *analyser {
	return &analyser{
		timeSeries: technical.NewTimeSeries(),
	}
}

func (a *analyser) addCandle(candle *Candle) {
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
	}

	fmt.Printf("last candle [%+v]\n", a.timeSeries.LastCandle().String())
}

func convertTechnicalCandle(candle *Candle) *technical.Candle {
	period := technical.TimePeriod{
		Start: candle.OpenTime,
		End:   candle.CloseTime,
	}

	timeSeriesCandle := technical.NewCandle(period)

	timeSeriesCandle.OpenPrice = big.NewFromString(candle.OpenPrice)
	timeSeriesCandle.ClosePrice = big.NewFromString(candle.ClosePrice)
	timeSeriesCandle.MaxPrice = big.NewFromString(candle.MaxPrice)
	timeSeriesCandle.MinPrice = big.NewFromString(candle.MinPrice)
	timeSeriesCandle.Volume = big.NewFromString(candle.Volume)
	timeSeriesCandle.TradeCount = candle.TradeCount

	return timeSeriesCandle
}

func areEqualTechnicalCandles(one, two *technical.Candle) bool {
	return one.Period.Start.Equal(two.Period.Start) &&
		one.Period.End.Equal(two.Period.End)
}
