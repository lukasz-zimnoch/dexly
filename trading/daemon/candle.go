package daemon

import (
	"context"
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading"
	"time"
)

const candleTickTimeout = 10 * time.Second

type CandleMonitor struct {
	logger     trading.Logger
	exchange   trading.ExchangeCandleService
	filter     *trading.CandleFilter
	repository trading.CandleRepository
	errChan    chan error
}

func RunCandleMonitor(
	ctx context.Context,
	logger trading.Logger,
	exchange trading.ExchangeCandleService,
	filter *trading.CandleFilter,
	repository trading.CandleRepository,
) *CandleMonitor {
	monitor := &CandleMonitor{
		logger:     logger,
		exchange:   exchange,
		filter:     filter,
		repository: repository,
		errChan:    make(chan error, 1),
	}

	go monitor.loop(ctx)

	return monitor
}

func (cm *CandleMonitor) loop(ctx context.Context) {
	candles, err := cm.exchange.Candles(ctx, cm.filter)
	if err != nil {
		cm.errChan <- fmt.Errorf("failed to get candles: [%v]", err)
		return
	}

	cm.logger.Debugf("fetched [%v] historical candles", len(candles))

	cm.repository.SaveCandles(candles...)

	tickTimeoutTimer := time.NewTimer(candleTickTimeout)
	ticker, tickerErrorChannel := cm.exchange.CandlesTicker(ctx, cm.filter)

	for {
		select {
		case tick := <-ticker:
			cm.logger.Debugf("received candle tick [%v]", tick)

			cm.repository.SaveCandles(tick.Candle)

			if !tickTimeoutTimer.Stop() {
				<-tickTimeoutTimer.C
			}
			tickTimeoutTimer.Reset(candleTickTimeout)
		case <-tickTimeoutTimer.C:
			cm.errChan <- fmt.Errorf("tick timeout expiration")
			return
		case err := <-tickerErrorChannel:
			cm.errChan <- fmt.Errorf("ticker error: [%v]", err)
			return
		case <-ctx.Done():
			return
		}
	}
}

func (cm *CandleMonitor) ErrChan() <-chan error {
	return cm.errChan
}
