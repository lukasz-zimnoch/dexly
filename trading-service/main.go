package main

import (
	"context"
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/exchange/binance"
	"time"
)

func main() {
	binanceClient := binance.NewClient()
	ticker, stopTicker, err := binanceClient.NewTicker("ETHUSDT")
	if err != nil {
		fmt.Println(err)
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	go func() {
		time.Sleep(1 * time.Minute)
		stopTicker()
		time.Sleep(1 * time.Minute)
		cancelCtx()
	}()

	for {
		select {
		case tick := <-ticker:
			fmt.Printf("[%+v]\n", tick)
		case <-ctx.Done():
			fmt.Printf("exiting program")
			return
		}
	}
}
