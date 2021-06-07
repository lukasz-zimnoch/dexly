package inmem

import (
	"github.com/lukasz-zimnoch/dexly/trading"
	"math/big"
	"sync"
)

type CandleRepository struct {
	candlesMutex sync.RWMutex
	candles      []*trading.Candle

	windowSize int
}

func NewCandleRepository(windowSize int) *CandleRepository {
	return &CandleRepository{
		candles:    make([]*trading.Candle, 0),
		windowSize: windowSize,
	}
}

func (cr *CandleRepository) SaveCandles(candles ...*trading.Candle) {
	cr.candlesMutex.Lock()
	defer cr.candlesMutex.Unlock()

	for _, candle := range candles {
		var lastCandle *trading.Candle
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
			if len(cr.candles) > cr.windowSize {
				index := 0
				copy(cr.candles[index:], cr.candles[index+1:])
				cr.candles[len(cr.candles)-1] = nil
				cr.candles = cr.candles[:len(cr.candles)-1]
			}
		}
	}
}

func (cr *CandleRepository) Candles() []*trading.Candle {
	cr.candlesMutex.RLock()
	defer cr.candlesMutex.RUnlock()

	snapshot := make([]*trading.Candle, len(cr.candles))
	copy(snapshot, cr.candles)

	return snapshot
}

func (cr *CandleRepository) LastClosePrice() (*big.Float, error) {
	cr.candlesMutex.RLock()
	defer cr.candlesMutex.RUnlock()

	price := new(big.Float)
	err := price.UnmarshalText([]byte(cr.candles[len(cr.candles)-1].ClosePrice))
	if err != nil {
		return nil, err
	}

	return price, nil
}
