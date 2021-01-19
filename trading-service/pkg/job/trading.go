package job

import (
	"context"
	"github.com/lukasz-zimnoch/dexly/trading-service/configs"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/trading"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/data/exchange/binance"
	"github.com/sirupsen/logrus"
	"time"
)

const tradingEnginesActivityCheckTick = 1 * time.Minute

func RunTrading(ctx context.Context, config *configs.Config) {
	logrus.Infof("running trading job")
	defer logrus.Infof("terminating trading job")

	tradingEngines := make([]*trading.Engine, 0)

	tradingEngines = append(
		tradingEngines,
		runBinanceTrading(ctx, &config.Binance),
	)

	ticker := time.NewTicker(tradingEnginesActivityCheckTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logrus.Infof("performing trading engines activity check")

			noActiveEngines := true

			for _, tradingEngine := range tradingEngines {
				if tradingEngine.ActiveTraders() > 0 {
					noActiveEngines = false
					break
				}
			}

			if noActiveEngines {
				logrus.Warningf("all trading engines are inactive")
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func runBinanceTrading(
	ctx context.Context,
	config *configs.Binance,
) *trading.Engine {
	exchange := binance.NewClient(config.ApiKey, config.SecretKey)
	engine := trading.NewEngine(exchange)

	for _, pair := range config.Pairs {
		engine.ActivateTrader(ctx, pair)
	}

	return engine
}
