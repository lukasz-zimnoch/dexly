package main

import (
	"context"
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading"
	"github.com/lukasz-zimnoch/dexly/trading/binance"
	"github.com/lukasz-zimnoch/dexly/trading/inmem"
	"github.com/lukasz-zimnoch/dexly/trading/logrus"
	"github.com/lukasz-zimnoch/dexly/trading/postgres"
	"github.com/lukasz-zimnoch/dexly/trading/pubsub"
	"github.com/lukasz-zimnoch/dexly/trading/techan"
	"github.com/lukasz-zimnoch/dexly/trading/uuid"
	"os"
)

// TODO: Metrics and monitoring.
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

	idService := &uuid.IDService{}

	pubsubClient, err := pubsub.NewClient(
		ctx,
		config.Pubsub.ProjectID,
		config.Pubsub.NotificationsTopicID,
	)
	if err != nil {
		logger.Fatalf("could not get pubsub client: [%v]", err)
	}

	trading.RunWorkloadController(
		ctx,
		postgres.NewWorkloadRepository(postgresClient, idService),
		idService,
		&exchangeConnector{},
		inmem.NewCandleRepository(trading.CandleWindowSize),
		techan.NewSignalGenerator(logger),
		postgres.NewPositionRepository(postgresClient, idService),
		postgres.NewOrderRepository(postgresClient, idService),
		pubsub.NewEventService(pubsubClient, logger),
		logger,
	)

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

type exchangeConnector struct{}

func (ec *exchangeConnector) Connect(
	ctx context.Context,
	workload *trading.Workload,
) (trading.ExchangeService, error) {
	switch workload.Account.Exchange {
	case "BINANCE":
		return binance.NewExchangeService(ctx, workload)
	default:
		panic("unknown exchange")
	}
}
