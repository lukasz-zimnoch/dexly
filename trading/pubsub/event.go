package pubsub

import "github.com/lukasz-zimnoch/dexly/trading"

type EventService struct{}

func NewEventService() *EventService {
	return &EventService{}
}

func (es *EventService) Publish(event *trading.Event) {}
