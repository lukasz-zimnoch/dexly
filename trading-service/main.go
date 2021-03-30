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

func init() {
	configureLogging()
}

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	configPath := os.Getenv("CONFIG")
	config, err := configs.ReadConfig(configPath)
	if err != nil {
		logrus.Fatalf("could not read config: [%v]", err)
	}

	if os.Getenv("DB_MIGRATION") == "on" {
		if err := runDatabaseMigration(&config.Database); err != nil {
			logrus.Fatalf("could not run database migration: [%v]", err)
		}
	}

	job.RunTrading(ctx, config)
}

func configureLogging() {
	if os.Getenv("LOG_FORMAT") == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	logLevel, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "could not parse log level: [%v]", err)
		os.Exit(1)
	}

	logrus.SetLevel(logLevel)

	logrus.SetOutput(os.Stdout)
}

func runDatabaseMigration(config *configs.Database) error {
	migrationsDir := "file://database/migrations"

	databaseAddress := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		config.User,
		config.Password,
		config.Address,
		config.Name,
	)

	migration, err := migrate.New(migrationsDir, databaseAddress)
	if err != nil {
		return err
	}

	return migration.Up()
}
