package main

import (
	"context"
	"fmt"
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
