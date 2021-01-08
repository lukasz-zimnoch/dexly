package job

import (
	"context"
	"github.com/lukasz-zimnoch/dexly/trading-service/configs"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/data/exchange/binance"
	log "github.com/sirupsen/logrus"
)

func RunTrading(ctx context.Context, config *configs.Config) {
	log.Infof("starting trading job")
	runBinanceTrading(ctx, &config.Binance)
}

func runBinanceTrading(ctx context.Context, config *configs.Binance) {
	exchange := binance.NewClient(config.ApiKey, config.SecretKey)
	engine := core.NewTradingEngine(exchange)

	for _, pair := range config.Pairs {
		engine.RunTrading(ctx, pair)
	}
}
