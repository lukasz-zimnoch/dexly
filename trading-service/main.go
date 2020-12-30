package main

import (
	"context"
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading-service/configs"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/job"
	log "github.com/sirupsen/logrus"
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
		log.Fatalf("could not read config")
	}

	log.Infof("starting trading job")

	job.RunTrading(ctx, config)

	<-ctx.Done()
}

func configureLogging() {
	if os.Getenv("LOG_FORMAT") == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
	}

	logLevel, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "could not parse log level")
		os.Exit(1)
	}

	log.SetLevel(logLevel)

	log.SetOutput(os.Stdout)
}
