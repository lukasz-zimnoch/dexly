package binance

import (
	"fmt"
	"github.com/adshao/go-binance"
)

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

type Tick struct {
	symbol    string
	timestamp int64
	price     string
}

type StopTicker func()

func (c *Client) NewTicker(symbol string) (chan *Tick, StopTicker, error) {
	const interval = "1m"

	tickChannel := make(chan *Tick)

	eventHandler := func(event *binance.WsKlineEvent) {
		tickChannel <- &Tick{
			symbol:    event.Symbol,
			timestamp: event.Time,
			price:     event.Kline.Close,
		}
	}

	errorHandler := func(err error) {
		fmt.Printf("received error: [%v]", err)
	}

	_, stopChannel, err := binance.WsKlineServe(
		symbol,
		interval,
		eventHandler,
		errorHandler,
	)
	if err != nil {
		return nil, nil, err
	}

	stopTicker := func() {
		stopChannel <- struct{}{}
	}

	return tickChannel, stopTicker, nil
}
