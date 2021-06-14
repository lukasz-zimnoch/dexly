package inmem

import (
	"github.com/lukasz-zimnoch/dexly/trading"
	"sync"
)

type CandleRepository struct {
	candlesMutex sync.RWMutex
	candles      map[string][]*trading.Candle

	windowSize int
}

func NewCandleRepository(windowSize int) *CandleRepository {
	return &CandleRepository{
		candles:    make(map[string][]*trading.Candle),
		windowSize: windowSize,
	}
}

func (cr *CandleRepository) SaveCandles(
	key string,
	candles ...*trading.Candle,
) {
	cr.candlesMutex.Lock()
	defer cr.candlesMutex.Unlock()

	for _, candle := range candles {
		var lastCandle *trading.Candle
		if len(cr.candles[key]) > 0 {
			lastCandle = cr.candles[key][len(cr.candles[key])-1]
		}

		if lastCandle != nil && lastCandle.Equal(candle) {
			lastCandle.OpenPrice = candle.OpenPrice
			lastCandle.ClosePrice = candle.ClosePrice
			lastCandle.MaxPrice = candle.MaxPrice
			lastCandle.MinPrice = candle.MinPrice
			lastCandle.Volume = candle.Volume
			lastCandle.TradeCount = candle.TradeCount
		} else {
			cr.candles[key] = append(cr.candles[key], candle)

			// remove oldest candle if window size has been exceeded
			if len(cr.candles[key]) > cr.windowSize {
				index := 0
				copy(cr.candles[key][index:], cr.candles[key][index+1:])
				cr.candles[key][len(cr.candles[key])-1] = nil
				cr.candles[key] = cr.candles[key][:len(cr.candles[key])-1]
			}
		}
	}
}

func (cr *CandleRepository) Candles(key string) []*trading.Candle {
	cr.candlesMutex.RLock()
	defer cr.candlesMutex.RUnlock()

	snapshot := make([]*trading.Candle, len(cr.candles[key]))
	copy(snapshot, cr.candles[key])

	return snapshot
}

func (cr *CandleRepository) DeleteCandles(key string) {
	cr.candlesMutex.RLock()
	defer cr.candlesMutex.RUnlock()

	delete(cr.candles, key)
}
