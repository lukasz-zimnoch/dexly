package trading

import (
	"fmt"
	"math/big"
)

type Signal struct {
	Type             PositionType
	EntryTarget      *big.Float
	TakeProfitTarget *big.Float
	StopLossTarget   *big.Float
}

func (s *Signal) String() string {
	return fmt.Sprintf(
		"%v, entry %v, tp: %v, sl: %v",
		s.Type.String(),
		s.EntryTarget.Text('f', 2),
		s.TakeProfitTarget.Text('f', 2),
		s.StopLossTarget.Text('f', 2),
	)
}

type SignalGenerator interface {
	Evaluate(candles []*Candle) (*Signal, bool)
}
