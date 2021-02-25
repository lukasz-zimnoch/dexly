package configs

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Database Database `yaml:"database"`
	Binance  Binance  `yaml:"binance"`
}

type Database struct {
	Address  string `yaml:"address"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

type Binance struct {
	ApiKey    string   `yaml:"apiKey"`
	SecretKey string   `yaml:"secretKey"`
	Pairs     []string `yaml:"pairs"`
}

func ReadConfig(configPath string) (*Config, error) {
	var config Config

	configYaml, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(configYaml, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
