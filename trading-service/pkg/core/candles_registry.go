package core

import (
	"sync"
)

type candlesRegistry struct {
	candlesMutex sync.RWMutex
	candles      []*Candle

	registrySize int
}

func newCandlesRegistry(registrySize int) *candlesRegistry {
	return &candlesRegistry{
		candles:      make([]*Candle, 0),
		registrySize: registrySize,
	}
}

func (cr *candlesRegistry) add(candles ...*Candle) {
	cr.candlesMutex.Lock()
	defer cr.candlesMutex.Unlock()

	for _, candle := range candles {
		var lastCandle *Candle
		if len(cr.candles) > 0 {
			lastCandle = cr.candles[len(cr.candles)-1]
		}

		if lastCandle != nil && lastCandle.Equal(candle) {
			lastCandle.OpenPrice = candle.OpenPrice
			lastCandle.ClosePrice = candle.ClosePrice
			lastCandle.MaxPrice = candle.MaxPrice
			lastCandle.MinPrice = candle.MinPrice
			lastCandle.Volume = candle.Volume
			lastCandle.TradeCount = candle.TradeCount
		} else {
			cr.candles = append(cr.candles, candle)

			// remove oldest candle if registry size has been exceeded
			if len(cr.candles) > cr.registrySize {
				index := 0
				copy(cr.candles[index:], cr.candles[index+1:])
				cr.candles[len(cr.candles)-1] = nil
				cr.candles = cr.candles[:len(cr.candles)-1]
			}
		}
	}
}

func (cr *candlesRegistry) get() []*Candle {
	cr.candlesMutex.RLock()
	defer cr.candlesMutex.RUnlock()

	snapshot := make([]*Candle, len(cr.candles))
	copy(snapshot, cr.candles)

	return snapshot
}
