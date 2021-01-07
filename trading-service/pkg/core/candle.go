package core

import (
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

type CandleTick struct {
	*Candle
	TickTime time.Time
}
