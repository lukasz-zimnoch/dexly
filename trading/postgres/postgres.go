package postgres

import (
	"context"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgtype"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lukasz-zimnoch/dexly/trading"
	"github.com/sirupsen/logrus"
	"math/big"
	"sync"
	"time"
)

type Config struct {
	Address      string
	User         string
	Password     string
	Name         string
	SSLMode      string
	MigrationDir string
}

type Client struct {
	mutex    sync.RWMutex
	database *sqlx.DB
}

func NewClient(ctx context.Context, config *Config) (*Client, error) {
	database, err := connectDatabase(config)
	if err != nil {
		return nil, err
	}

	client := &Client{database: database}

	go client.monitorDatabaseMode(ctx, config)

	return client, nil
}

func connectDatabase(config *Config) (*sqlx.DB, error) {
	address := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		config.User,
		config.Password,
		config.Address,
		config.Name,
		config.SSLMode,
	)

	database, err := sqlx.Connect("pgx", address)
	if err != nil {
		return nil, fmt.Errorf("could not connect database: [%v]", err)
	}

	return database, nil
}

func (c *Client) monitorDatabaseMode(ctx context.Context, config *Config) {
	ticker := time.NewTicker(1 * time.Minute)

	for {
		select {
		case <-ticker.C:
			var isReadonly bool
			err := c.database.Get(&isReadonly, "SELECT pg_is_in_recovery()")
			if err != nil {
				logrus.Errorf(
					"could not determine database mode: [%v]",
					err,
				)
				continue
			}

			if isReadonly {
				logrus.Infof(
					"database instance demoted to read-only mode; " +
						"reconnecting master database",
				)

				newDatabase, err := connectDatabase(config)
				if err != nil {
					logrus.Errorf(
						"could not reconnect master database: [%v]",
						err,
					)
					continue
				}

				c.mutex.Lock()
				_ = c.database.Close()
				c.database = newDatabase
				c.mutex.Unlock()

				logrus.Infof("reconnected master database")
			}
		case <-ctx.Done():
			_ = c.database.Close()
			return
		}
	}
}

func (c *Client) instance() *sqlx.DB {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.database
}

func RunMigration(
	logger trading.Logger,
	config *Config,
) error {
	if len(config.MigrationDir) == 0 {
		logger.Infof("postgres migration disabled")
		return nil
	}

	logger.Infof("starting postgres migration")

	migrationsDir := "file://" + config.MigrationDir

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
			logger.Infof("postgres migration skipped as there are no changes")
			return nil
		}

		return err
	}

	logger.Infof("postgres migration performed successfully")

	return nil
}

func floatToNumeric(value *big.Float) (pgtype.Numeric, error) {
	var result pgtype.Numeric
	valueFloat, _ := value.Float64()

	if err := result.Set(valueFloat); err != nil {
		return pgtype.Numeric{}, err
	}

	return result, nil
}

func numericToFloat(value pgtype.Numeric) (*big.Float, error) {
	var result float64

	if err := value.AssignTo(&result); err != nil {
		return nil, err
	}

	return big.NewFloat(result), nil
}
