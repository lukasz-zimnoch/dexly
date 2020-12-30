package job

import (
	"context"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/data/exchange/binance"
)

func RunTrading(ctx context.Context) {
	engine := core.NewTradingEngine(binance.NewClient())
	engine.Observe(ctx, "ETHUSDT")
}
