package pubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"github.com/lukasz-zimnoch/dexly/trading"
)

type EventService struct {
	client *Client
	logger trading.Logger
}

func NewEventService(client *Client, logger trading.Logger) *EventService {
	return &EventService{client, logger}
}

func (es *EventService) Publish(event *trading.Event) {
	es.publishOnNotificationsTopic(context.TODO(), event)
}

func (es *EventService) publishOnNotificationsTopic(
	ctx context.Context,
	event *trading.Event,
) {
	topicLogger := es.logger.WithField("topic", "notifications")

	messageData, err := json.Marshal(&notificationEvent{
		Email:   event.Account.Email,
		Payload: event.Payload,
	})
	if err != nil {
		topicLogger.Errorf("could not marshal trading event: [%v]", err)
		return
	}

	es.publishOnTopic(
		ctx,
		es.client.notificationsTopic,
		messageData,
		topicLogger,
	)
}

func (es *EventService) publishOnTopic(
	ctx context.Context,
	topic *pubsub.Topic,
	messageData []byte,
	topicLogger trading.Logger,
) {
	result := topic.Publish(ctx, &pubsub.Message{
		Data: messageData,
	})

	go func() {
		id, err := result.Get(ctx)
		if err != nil {
			topicLogger.Errorf(
				"could not publish trading event: [%v]",
				err,
			)
			return
		}

		topicLogger.Infof("published trading event with ID: [%v]", id)
	}()
}

type notificationEvent struct {
	Email   string
	Payload string
}
