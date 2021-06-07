package techan

import (
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading"
	techanbig "github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
	"math/big"
	"strings"
	"time"
)

const strategyBackoff = 5 * time.Minute

// TODO: get rid of the `techan` library
type SignalGenerator struct {
	logger           trading.Logger
	pair             trading.Pair
	candleRepository trading.CandleRepository
	lastSignalTime   time.Time
}

func NewSignalGenerator(
	logger trading.Logger,
	pair trading.Pair,
	candleRepository trading.CandleRepository,
) *SignalGenerator {
	return &SignalGenerator{
		logger:           logger,
		pair:             pair,
		candleRepository: candleRepository,
		lastSignalTime:   time.Now(),
	}
}

func (sg *SignalGenerator) Poll() (*trading.Signal, bool) {
	if time.Now().Before(sg.lastSignalTime.Add(strategyBackoff)) {
		return nil, false
	}

	candles := techan.NewTimeSeries()

	for _, currentCandle := range sg.candleRepository.Candles() {
		candles.AddCandle(toTechanCandle(currentCandle))
	}

	lastIndex := candles.LastIndex()
	price := techan.NewClosePriceIndicator(candles)
	priceEma := techan.NewEMAIndicator(price, 50)
	entryRule := newNearCrossUpIndicatorRule(priceEma, price)

	sg.logIndicators(price, priceEma, lastIndex)

	// Check against the second to last index because the last index is not
	// yet stable as its value changes.
	if entryRule.IsSatisfied(lastIndex-1, nil) {
		entryTarget := big.NewFloat(
			price.Calculate(lastIndex).Float(),
		)

		priceChangeFactor := 0.025 // TODO: use ATR indicator
		stopLossFactor := big.NewFloat(1 - priceChangeFactor)
		takeProfitFactor := big.NewFloat(1 + (2 * priceChangeFactor))

		stopLossTarget := new(big.Float).Mul(entryTarget, stopLossFactor)
		takeProfitTarget := new(big.Float).Mul(entryTarget, takeProfitFactor)

		sg.lastSignalTime = time.Now()

		return &trading.Signal{
			Type:             trading.LONG,
			EntryTarget:      entryTarget,
			TakeProfitTarget: takeProfitTarget,
			StopLossTarget:   stopLossTarget,
		}, true
	}

	return nil, false
}

func toTechanCandle(candle *trading.Candle) *techan.Candle {
	period := techan.TimePeriod{
		Start: candle.OpenTime,
		End:   candle.CloseTime,
	}

	techanCandle := techan.NewCandle(period)

	techanCandle.OpenPrice = techanbig.NewFromString(candle.OpenPrice)
	techanCandle.ClosePrice = techanbig.NewFromString(candle.ClosePrice)
	techanCandle.MaxPrice = techanbig.NewFromString(candle.MaxPrice)
	techanCandle.MinPrice = techanbig.NewFromString(candle.MinPrice)
	techanCandle.Volume = techanbig.NewFromString(candle.Volume)
	techanCandle.TradeCount = candle.TradeCount

	return techanCandle
}

type nearCrossRule struct {
	upper techan.Indicator
	lower techan.Indicator
	cmp   int
}

func newNearCrossUpIndicatorRule(
	upper, lower techan.Indicator,
) techan.Rule {
	return nearCrossRule{
		upper: upper,
		lower: lower,
		cmp:   1,
	}
}

func (ncr nearCrossRule) IsSatisfied(
	index int,
	_ *techan.TradingRecord,
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

func (sg *SignalGenerator) logIndicators(
	price,
	priceEma techan.Indicator,
	lastIndex int,
) {
	indexes := []int{lastIndex, lastIndex - 1, lastIndex - 2}

	sg.logger.Debugf(
		"price: %v, ema: %v",
		stringifyIndicator(price, indexes),
		stringifyIndicator(priceEma, indexes),
	)
}

func stringifyIndicator(indicator techan.Indicator, indexes []int) string {
	components := make([]string, 0)

	for _, index := range indexes {
		components = append(
			components,
			fmt.Sprintf(
				"%v=%v",
				index,
				indicator.Calculate(index).FormattedString(2),
			),
		)
	}

	return strings.Join(components, " ")
}
