package trading

import (
	"fmt"
	"math/big"
)

type Signal struct {
	Pair             Pair
	Type             PositionType
	EntryTarget      *big.Float
	TakeProfitTarget *big.Float
	StopLossTarget   *big.Float
}

func (s *Signal) String() string {
	return fmt.Sprintf(
		"%v (%v), entry %v, tp: %v, sl: %v",
		s.Pair.String(),
		s.Type.String(),
		s.EntryTarget.Text('f', 2),
		s.TakeProfitTarget.Text('f', 2),
		s.StopLossTarget.Text('f', 2),
	)
}

type SignalGenerator interface {
	Poll() (*Signal, bool)
}
