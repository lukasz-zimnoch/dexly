package trading

type Event struct{}

type EventService interface {
	Publish(event *Event)
}
