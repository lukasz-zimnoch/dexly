package core

import (
	"context"
	"fmt"
	"time"
)

const tickTimeout = 10 * time.Second

type candlesSink interface {
	add(candles ...*Candle)
}

type candlesMonitor struct {
	exchange ExchangeClient
}

func newCandlesMonitor(
	exchange ExchangeClient,
) *candlesMonitor {
	return &candlesMonitor{
		exchange: exchange,
	}
}

func (cm *candlesMonitor) run(
	ctx context.Context,
	filter *CandlesFilter,
	sink candlesSink,
) <-chan error {
	errorChannel := make(chan error)

	go func() {
		candles, err := cm.exchange.Candles(ctx, filter)
		if err != nil {
			errorChannel <- fmt.Errorf("failed to get candles: [%v]", err)
		}

		sink.add(candles...)

		tickTimeoutTimer := time.NewTimer(tickTimeout)
		ticker, tickerErrorChannel := cm.exchange.CandlesTicker(ctx, filter)

		for {
			select {
			case tick := <-ticker:
				sink.add(tick.Candle)

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

	return errorChannel
}
