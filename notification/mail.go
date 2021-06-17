package notification

import (
	"fmt"
	"gopkg.in/mail.v2"
)

type MailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

type MailService struct {
	config *MailConfig
}

func NewMailService(auth *MailConfig) *MailService {
	return &MailService{auth}
}

func (ms *MailService) Send(notification *Notification) error {
	message := mail.NewMessage()
	message.SetHeader("From", ms.config.Username)
	message.SetHeader("To", notification.AccountEmail)
	message.SetHeader("Subject", "Dexly notification")
	message.SetBody("text/plain", notification.Payload)

	dialer := mail.NewDialer(
		ms.config.Host,
		ms.config.Port,
		ms.config.Username,
		ms.config.Password,
	)

	if err := dialer.DialAndSend(message); err != nil {
		return fmt.Errorf("could not send email: [%v]", err)
	}

	return nil
}
