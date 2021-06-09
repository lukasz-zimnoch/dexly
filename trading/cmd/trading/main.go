package main

import (
	"context"
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading"
	"github.com/lukasz-zimnoch/dexly/trading/binance"
	"github.com/lukasz-zimnoch/dexly/trading/daemon"
	"github.com/lukasz-zimnoch/dexly/trading/inmem"
	"github.com/lukasz-zimnoch/dexly/trading/logrus"
	"github.com/lukasz-zimnoch/dexly/trading/postgres"
	"github.com/lukasz-zimnoch/dexly/trading/techan"
	"os"
)

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	config, err := readConfig()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "could not read config: [%v]", err)
		os.Exit(1)
	}

	logger := logrus.ConfigureStandardLogger(
		config.Logging.Format,
		config.Logging.Level,
	)

	postgresClient, err := connectPostgres(ctx, logger, &config.Database)
	if err != nil {
		logger.Fatalf("could not connect postgres: [%v]", err)
	}

	binanceExchangeService, err := binance.NewExchangeService(
		ctx,
		config.Binance.ApiKey,
		config.Binance.SecretKey,
		config.Binance.Testnet,
	)
	if err != nil {
		logger.Fatalf("could not create binance handle: [%v]", err)
	}

	workerController, err := daemon.RunWorkerController(
		logger,
		inmem.NewAccountRepository(),
		func(windowSize int) trading.CandleRepository {
			return inmem.NewCandleRepository(windowSize)
		},
		func(
			logger trading.Logger,
			pair trading.Pair,
			candleRepository trading.CandleRepository,
		) trading.SignalGenerator {
			return techan.NewSignalGenerator(logger, pair, candleRepository)
		},
		postgres.NewPositionRepository(postgresClient),
		postgres.NewOrderRepository(postgresClient),
		binanceExchangeService,
	)
	if err != nil {
		logger.Fatalf("could not run worker controller: [%v]", err)
	}

	for _, pair := range config.Binance.Pairs {
		workerController.ActivateWorker(ctx, trading.ParsePair(pair))
	}

	<-ctx.Done()
}

func connectPostgres(
	ctx context.Context,
	logger trading.Logger,
	config *Database,
) (*postgres.Client, error) {
	if err := postgres.RunMigration(
		logger,
		(*postgres.Config)(config),
	); err != nil {
		return nil, fmt.Errorf(
			"could not run postgres migration: [%v]",
			err,
		)
	}

	client, err := postgres.NewClient(
		ctx,
		(*postgres.Config)(config),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"could not create postgres client: [%v]",
			err,
		)
	}

	return client, nil
}
