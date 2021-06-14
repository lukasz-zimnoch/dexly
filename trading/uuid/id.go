package uuid

import (
	"github.com/google/uuid"
	"github.com/lukasz-zimnoch/dexly/trading"
)

type IDService struct{}

func (ids *IDService) NewID() trading.ID {
	return uuid.New()
}

func (ids *IDService) NewIDFromString(id string) (trading.ID, error) {
	return uuid.Parse(id)
}
