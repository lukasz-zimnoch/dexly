package main

import (
	"context"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lukasz-zimnoch/dexly/trading-service/configs"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/job"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	config, err := configs.ReadConfig()
	if err != nil {
		logrus.Fatalf("could not read config: [%v]", err)
	}

	configureLogging(&config.Logging)

	if config.Database.Migration {
		if err := runDatabaseMigration(&config.Database); err != nil {
			logrus.Fatalf("could not run database migration: [%v]", err)
		}
	}

	job.RunTrading(ctx, config)
}

func configureLogging(config *configs.Logging) {
	if config.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	logLevel, err := logrus.ParseLevel(config.Level)
	if err != nil {
		logrus.Fatalf("could not parse log level: [%v]", err)
	}

	logrus.SetLevel(logLevel)

	logrus.SetOutput(os.Stdout)
}

func runDatabaseMigration(config *configs.Database) error {
	logrus.Infof("starting database migration")

	migrationsDir := "file://database/migrations"

	databaseAddress := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		config.User,
		config.Password,
		config.Address,
		config.Name,
		config.SSLMode,
	)

	migration, err := migrate.New(migrationsDir, databaseAddress)
	if err != nil {
		return err
	}

	err = migration.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			logrus.Infof("database migration skipped as there are no changes")
			return nil
		}

		return err
	}

	logrus.Infof("database migration performed successfully")

	return nil
}
