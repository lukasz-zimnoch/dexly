package candle

import (
	"sync"
)

type Registry struct {
	candlesMutex sync.RWMutex
	candles      []*Candle

	registrySize int
}

func NewRegistry(registrySize int) *Registry {
	return &Registry{
		candles:      make([]*Candle, 0),
		registrySize: registrySize,
	}
}

func (r *Registry) AddCandles(candles ...*Candle) {
	r.candlesMutex.Lock()
	defer r.candlesMutex.Unlock()

	for _, candle := range candles {
		var lastCandle *Candle
		if len(r.candles) > 0 {
			lastCandle = r.candles[len(r.candles)-1]
		}

		if lastCandle != nil && lastCandle.Equal(candle) {
			lastCandle.OpenPrice = candle.OpenPrice
			lastCandle.ClosePrice = candle.ClosePrice
			lastCandle.MaxPrice = candle.MaxPrice
			lastCandle.MinPrice = candle.MinPrice
			lastCandle.Volume = candle.Volume
			lastCandle.TradeCount = candle.TradeCount
		} else {
			r.candles = append(r.candles, candle)

			// remove oldest candle if registry size has been exceeded
			if len(r.candles) > r.registrySize {
				index := 0
				copy(r.candles[index:], r.candles[index+1:])
				r.candles[len(r.candles)-1] = nil
				r.candles = r.candles[:len(r.candles)-1]
			}
		}
	}
}

func (r *Registry) Candles() []*Candle {
	r.candlesMutex.RLock()
	defer r.candlesMutex.RUnlock()

	snapshot := make([]*Candle, len(r.candles))
	copy(snapshot, r.candles)

	return snapshot
}
