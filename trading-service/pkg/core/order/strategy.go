package order

import (
	technical "github.com/sdcoffey/techan"
)

// TODO: implement proper strategy
func evaluateStrategy(series *technical.TimeSeries) technical.Strategy {
	price := technical.NewClosePriceIndicator(series)
	priceEma := technical.NewEMAIndicator(price, 100)

	entryRule := technical.And(
		newNearCrossUpIndicatorRule(priceEma, price),
		technical.PositionNewRule{})

	exitPrice := technical.NewConstantIndicator(1100)

	exitRule := technical.And(
		technical.NewCrossDownIndicatorRule(price, exitPrice),
		technical.PositionOpenRule{})

	return technical.RuleStrategy{
		EntryRule: entryRule,
		ExitRule:  exitRule,
	}
}

type nearCrossRule struct {
	upper technical.Indicator
	lower technical.Indicator
	cmp   int
}

func newNearCrossUpIndicatorRule(
	upper, lower technical.Indicator,
) technical.Rule {
	return nearCrossRule{
		upper: upper,
		lower: lower,
		cmp:   1,
	}
}

func (ncr nearCrossRule) IsSatisfied(
	index int,
	record *technical.TradingRecord,
) bool {
	if index == 0 {
		return false
	}

	current := ncr.lower.Calculate(index).
		Cmp(ncr.upper.Calculate(index))

	previous := ncr.lower.Calculate(index - 1).
		Cmp(ncr.upper.Calculate(index - 1))

	if (current == 0 || current == ncr.cmp) &&
		(previous == 0 || previous == -ncr.cmp) {
		return true
	}

	return false
}
