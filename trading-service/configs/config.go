package configs

import (
	"github.com/sherifabdlnaby/configuro"
)

// Config values can be set using either environment variables with `CONFIG_`
// prefix or config.yml file placed in working directory.
// See https://github.com/sherifabdlnaby/configuro.
type Config struct {
	Logging  Logging
	Database Database
	Binance  Binance
}

type Logging struct {
	Level  string
	Format string
}

type Database struct {
	Address   string
	User      string
	Password  string
	Name      string
	Migration bool
}

type Binance struct {
	ApiKey    string
	SecretKey string
	Pairs     []string
}

func ReadConfig() (*Config, error) {
	loader, err := configuro.NewConfig()
	if err != nil {
		return nil, err
	}

	// Default config values.
	config := &Config{
		Logging: Logging{
			Level: "info",
		},
		Database: Database{
			Address:  "localhost:5432",
			User:     "postgres",
			Password: "postgres",
			Name:     "postgres",
		},
	}

	err = loader.Load(config)
	if err != nil {
		return nil, err
	}

	err = loader.Validate(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
