package strategy

import (
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/candle"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/trade"
)

type candleSupplier interface {
	Candles() []*candle.Candle
}

type EmaCross struct {
	candleSupplier candleSupplier
}

func NewStrategy(candleSupplier candleSupplier) *EmaCross {
	return &EmaCross{
		candleSupplier: candleSupplier,
	}
}

func (ec *EmaCross) Evaluate() (*trade.Signal, bool) {
	return nil, false
}
