package core

import (
	"github.com/sdcoffey/big"
	technical "github.com/sdcoffey/techan"
	log "github.com/sirupsen/logrus"
)

type analyser struct {
	timeSeries *technical.TimeSeries
}

func newAnalyser() *analyser {
	return &analyser{
		timeSeries: technical.NewTimeSeries(),
	}
}

// TODO: remove the oldest candle
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

	// FIXME: temporary debug log
	newLastCandle := a.timeSeries.LastCandle()
	log.WithFields(log.Fields{
		"period": newLastCandle.Period.String(),
		"close":  newLastCandle.ClosePrice,
		"volume": newLastCandle.Volume,
	}).Debugf("candle update registered")
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
