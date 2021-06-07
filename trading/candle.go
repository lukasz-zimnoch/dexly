package trading

import (
	"fmt"
	"math/big"
	"time"
)

type Candle struct {
	Pair       string
	Exchange   string
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

type CandleFilter struct {
	Pair      string
	Interval  string
	StartTime time.Time
	EndTime   time.Time
}

type CandleTick struct {
	*Candle
	TickTime time.Time
}

func (ct *CandleTick) String() string {
	return ct.Candle.String()
}

type CandleRepository interface {
	SaveCandles(candles ...*Candle)

	Candles() []*Candle

	LastClosePrice() (*big.Float, error)
}
