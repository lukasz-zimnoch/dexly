package candle

import (
	"context"
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/logger"
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

func (c *Candle) String() string {
	return fmt.Sprintf(
		"time: %v, price: %v",
		c.OpenTime.Format(time.RFC3339),
		c.ClosePrice,
	)
}

type Tick struct {
	*Candle
	TickTime time.Time
}

func (t *Tick) String() string {
	return t.Candle.String()
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

func RunMonitor(
	ctx context.Context,
	logger logger.Logger,
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

		logger.Debugf("fetched [%v] historical candles", len(candles))

		sink.Add(candles...)

		tickTimeoutTimer := time.NewTimer(tickTimeout)
		ticker, tickerErrorChannel := provider.CandlesTicker(ctx, filter)

		for {
			select {
			case tick := <-ticker:
				logger.Debugf("received candle tick [%v]", tick)

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
