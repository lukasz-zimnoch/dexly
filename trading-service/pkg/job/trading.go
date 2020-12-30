package job

import (
	"context"
	"github.com/lukasz-zimnoch/dexly/trading-service/configs"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/data/exchange/binance"
	log "github.com/sirupsen/logrus"
)

func RunTrading(ctx context.Context, config *configs.Config) {
	log.Infof("creating trading engine")

	exchange := binance.NewClient(config.Binance.ApiKey, config.Binance.SecretKey)
	engine := core.NewTradingEngine(exchange)
	engine.Observe(ctx, "ETHUSDT") // TODO: symbols from config
}
