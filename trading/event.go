package trading

import (
	"fmt"
)

type Event struct {
	Account *Account
	Payload string
}

func NewPositionOpenedEvent(workload *Workload, position *Position) *Event {
	return &Event{
		Account: workload.Account,
		Payload: fmt.Sprintf(
			"New position has been opened:\n"+
				"- ID: %v\n"+
				"- Exchange: %v\n"+
				"- Pair: %v\n"+
				"- Size: %v\n"+
				"- Entry price: %v\n"+
				"- Take profit price: %v\n"+
				"- Stop loss price: %v",
			position.ID.String(),
			workload.Account.Exchange,
			string(workload.Pair.Symbol()),
			position.Size.Text('f', 2),
			position.EntryPrice.Text('f', 2),
			position.TakeProfitPrice.Text('f', 2),
			position.StopLossPrice.Text('f', 2),
		),
	}
}

func NewPositionClosedEvent(workload *Workload, position *Position) *Event {
	return &Event{
		Account: workload.Account,
		Payload: fmt.Sprintf(
			"Position has been closed:\n"+
				"- ID: %v\n"+
				"- Exchange: %v\n"+
				"- Pair: %v\n",
			position.ID.String(),
			workload.Account.Exchange,
			string(workload.Pair.Symbol()),
		),
	}
}

type EventService interface {
	Publish(event *Event)
}
