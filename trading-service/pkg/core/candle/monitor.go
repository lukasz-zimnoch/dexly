package candle

import (
	"context"
	"fmt"
	"time"
)

const tickTimeout = 10 * time.Second

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

type Tick struct {
	*Candle
	TickTime time.Time
}

type Filter struct {
	Pair      string
	Interval  string
	StartTime time.Time
	EndTime   time.Time
}

type Provider interface {
	Candles(
		ctx context.Context,
		filter *Filter,
	) ([]*Candle, error)

	CandlesTicker(
		ctx context.Context,
		filter *Filter,
	) (<-chan *Tick, <-chan error)
}

type Sink interface {
	Add(candles ...*Candle)
}

type Monitor struct {
	ErrorChannel <-chan error
}

// TODO: add logging
func RunMonitor(
	ctx context.Context,
	provider Provider,
	filter *Filter,
	sink Sink,
) *Monitor {
	errorChannel := make(chan error, 1)

	go func() {
		candles, err := provider.Candles(ctx, filter)
		if err != nil {
			errorChannel <- fmt.Errorf("failed to get candles: [%v]", err)
			return
		}

		sink.Add(candles...)

		tickTimeoutTimer := time.NewTimer(tickTimeout)
		ticker, tickerErrorChannel := provider.CandlesTicker(ctx, filter)

		for {
			select {
			case tick := <-ticker:
				sink.Add(tick.Candle)

				if !tickTimeoutTimer.Stop() {
					<-tickTimeoutTimer.C
				}
				tickTimeoutTimer.Reset(tickTimeout)
			case <-tickTimeoutTimer.C:
				errorChannel <- fmt.Errorf("tick timeout expiration")
				return
			case err := <-tickerErrorChannel:
				errorChannel <- fmt.Errorf("ticker error: [%v]", err)
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return &Monitor{
		ErrorChannel: errorChannel,
	}
}
