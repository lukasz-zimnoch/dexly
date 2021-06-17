package trading

type Event struct {
	AccountEmail string
	Payload      string
}

type EventService interface {
	Publish(event *Event)
}
