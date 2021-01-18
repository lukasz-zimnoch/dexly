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
	errorChannel <-chan error
}

func runCandlesMonitor(
	ctx context.Context,
	exchange ExchangeClient,
	filter *CandlesFilter,
	candlesSink candlesSink,
) *candlesMonitor {
	errorChannel := make(chan error)

	go func() {
		candles, err := exchange.Candles(ctx, filter)
		if err != nil {
			errorChannel <- fmt.Errorf("failed to get candles: [%v]", err)
		}

		candlesSink.add(candles...)

		tickTimeoutTimer := time.NewTimer(tickTimeout)
		ticker, tickerErrorChannel := exchange.CandlesTicker(ctx, filter)

		for {
			select {
			case tick := <-ticker:
				candlesSink.add(tick.Candle)

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

	return &candlesMonitor{
		errorChannel: errorChannel,
	}
}
