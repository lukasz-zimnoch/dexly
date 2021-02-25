package job

import (
	"context"
	"github.com/lukasz-zimnoch/dexly/trading-service/configs"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/trade"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/worker"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/data/exchange/binance"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/data/postgres"
	pgtrade "github.com/lukasz-zimnoch/dexly/trading-service/pkg/data/postgres/trade"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

const workerEnginesActivityCheckTick = 1 * time.Minute

func RunTrading(ctx context.Context, config *configs.Config) {
	logrus.Infof("running trading job")
	defer logrus.Infof("terminating trading job")

	tradeRepository, err := connectTradeRepository(ctx, &config.Database)
	if err != nil {
		logrus.Errorf("could not connect trade repository: [%v]", err)
		return
	}

	engines := make([]*worker.Engine, 0)

	engines = append(
		engines,
		runBinanceEngine(ctx, &config.Binance, tradeRepository),
	)

	ticker := time.NewTicker(workerEnginesActivityCheckTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logrus.Infof("performing worker engines activity check")

			noActiveEngines := true

			for _, engine := range engines {
				if engine.ActiveWorkers() > 0 {
					noActiveEngines = false
					break
				}
			}

			if noActiveEngines {
				logrus.Warningf("all worker engines are inactive")
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func connectTradeRepository(
	ctx context.Context,
	config *configs.Database,
) (trade.Repository, error) {
	client, err := postgres.NewClient(
		ctx,
		&postgres.Config{
			Address:  config.Address,
			User:     config.User,
			Password: config.Password,
			Name:     config.Name,
		},
	)
	if err != nil {
		return nil, err
	}

	return pgtrade.NewPgRepository(client), nil
}

func runBinanceEngine(
	ctx context.Context,
	config *configs.Binance,
	tradeRepository trade.Repository,
) *worker.Engine {
	exchange := binance.NewClient(config.ApiKey, config.SecretKey)
	engine := worker.NewEngine(exchange, tradeRepository)

	for _, pair := range config.Pairs {
		engine.ActivateWorker(ctx, parsePair(pair))
	}

	return engine
}

func parsePair(pair string) worker.Pair {
	symbols := strings.Split(pair, "/")

	return worker.Pair{
		Base:  symbols[0],
		Quote: symbols[1],
	}
}
