package trading

import (
	"fmt"
	"time"
)

// For the time being, we always use the 1m interval and a 12h window size.
const (
	CandleInterval   = "1m"
	CandleWindowSize = 720
)

type Candle struct {
	OpenTime   time.Time
	CloseTime  time.Time
	OpenPrice  string
	ClosePrice string
	MaxPrice   string
	MinPrice   string
	Volume     string
	TradeCount uint
}

func (c *Candle) Equal(other *Candle) bool {
	return c.OpenTime.Equal(other.OpenTime) &&
		c.CloseTime.Equal(other.CloseTime)
}

func (c *Candle) String() string {
	return fmt.Sprintf(
		"time: %v, price: %v",
		c.OpenTime.Format(time.RFC3339),
		c.ClosePrice,
	)
}

type CandleTick struct {
	*Candle
	TickTime time.Time
}

func (ct *CandleTick) String() string {
	return ct.Candle.String()
}

type CandleRepository interface {
	SaveCandles(key string, candles ...*Candle)

	Candles(key string) []*Candle

	DeleteCandles(key string)
}
