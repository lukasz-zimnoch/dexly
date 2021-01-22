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

type Monitor struct {
	logger   logger.Logger
	provider Provider
	filter   *Filter
	registry *Registry
	errChan  chan error
}

func RunMonitor(
	ctx context.Context,
	logger logger.Logger,
	provider Provider,
	filter *Filter,
	registry *Registry,
) *Monitor {
	monitor := &Monitor{
		logger:   logger,
		provider: provider,
		filter:   filter,
		registry: registry,
		errChan:  make(chan error, 1),
	}

	go monitor.loop(ctx)

	return monitor
}

func (m *Monitor) loop(ctx context.Context) {
	candles, err := m.provider.Candles(ctx, m.filter)
	if err != nil {
		m.errChan <- fmt.Errorf("failed to get candles: [%v]", err)
		return
	}

	m.logger.Debugf("fetched [%v] historical candles", len(candles))

	m.registry.AddCandles(candles...)

	tickTimeoutTimer := time.NewTimer(tickTimeout)
	ticker, tickerErrorChannel := m.provider.CandlesTicker(ctx, m.filter)

	for {
		select {
		case tick := <-ticker:
			m.logger.Debugf("received candle tick [%v]", tick)

			m.registry.AddCandles(tick.Candle)

			if !tickTimeoutTimer.Stop() {
				<-tickTimeoutTimer.C
			}
			tickTimeoutTimer.Reset(tickTimeout)
		case <-tickTimeoutTimer.C:
			m.errChan <- fmt.Errorf("tick timeout expiration")
			return
		case err := <-tickerErrorChannel:
			m.errChan <- fmt.Errorf("ticker error: [%v]", err)
			return
		case <-ctx.Done():
			return
		}
	}
}

func (m *Monitor) ErrChan() <-chan error {
	return m.errChan
}
