package trading

import "fmt"

type ID interface {
	fmt.Stringer
}

type IDService interface {
	NewID() ID

	NewIDFromString(id string) (ID, error)
}
