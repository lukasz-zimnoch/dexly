package postgres

import (
	"context"
	"fmt"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
)

type Config struct {
	Address  string
	User     string
	Password string
	Name     string
}

type Client struct {
	*sqlx.DB
}

func NewClient(ctx context.Context, config *Config) (*Client, error) {
	address := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		config.User,
		config.Password,
		config.Address,
		config.Name,
	)

	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		return nil, fmt.Errorf("could not connect database: [%v]", err)
	}

	go func() {
		<-ctx.Done()
		_ = db.Close()
	}()

	return &Client{db}, nil
}
