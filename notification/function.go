package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

const (
	EnvMailHost     = "MAIL_HOST"
	EnvMailPort     = "MAIL_PORT"
	EnvMailUsername = "MAIL_USERNAME"
	EnvMailPassword = "MAIL_PASSWORD"
)

const (
	DefaultMailHost     = "smtp.gmail.com"
	DefaultMailPort     = "587"
	DefaultMailUsername = "dexly.bot@gmail.com"
)

var (
	mailService *MailService
)

type PubSubMessage struct {
	Data []byte `json:"data"`
}

func ProcessEvent(ctx context.Context, message PubSubMessage) error {
	if err := initializeMailService(); err != nil {
		return fmt.Errorf("could not initialize mail service: [%v]", err)
	}

	var event Event

	err := json.Unmarshal(message.Data, &event)
	if err != nil {
		return fmt.Errorf("could not unmarshal pubsub message: [%v]", err)
	}

	err = mailService.ProcessEvent(&event)
	if err != nil {
		return fmt.Errorf("mail service error: [%v]", err)
	}

	return nil
}

func initializeMailService() error {
	port, err := strconv.Atoi(getEnvOrDefault(EnvMailPort, DefaultMailPort))
	if err != nil {
		return fmt.Errorf("could not get port number: [%v]", err)
	}

	password := os.Getenv(EnvMailPassword)
	if len(password) == 0 {
		return fmt.Errorf("%v must be set", EnvMailPassword)
	}

	if mailService == nil {
		fmt.Println("initializing mail service instance")

		mailService = NewMailService(
			&MailConfig{
				Host:     getEnvOrDefault(EnvMailHost, DefaultMailHost),
				Port:     port,
				Username: getEnvOrDefault(EnvMailUsername, DefaultMailUsername),
				Password: password,
			},
		)
	}

	return nil
}

func getEnvOrDefault(envName, defaultValue string) string {
	if envValue := os.Getenv(envName); len(envValue) > 0 {
		return envValue
	}

	return defaultValue
}
