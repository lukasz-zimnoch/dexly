package migration

import (
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lukasz-zimnoch/dexly/trading"
	"github.com/lukasz-zimnoch/dexly/trading/postgres"
)

func RunPostgresMigration(
	logger trading.Logger,
	config *postgres.Config,
) error {
	logger.Infof("starting database migration")

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
			logger.Infof("database migration skipped as there are no changes")
			return nil
		}

		return err
	}

	logger.Infof("database migration performed successfully")

	return nil
}
