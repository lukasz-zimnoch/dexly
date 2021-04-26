package strategy

import (
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/candle"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/trade"
	techanbig "github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
	"math/big"
)

type candleSupplier interface {
	Candles() []*candle.Candle
}

// TODO: get rid of the `techan` library
type EmaCross struct {
	candleSupplier candleSupplier
}

func NewEmaCross(candleSupplier candleSupplier) *EmaCross {
	return &EmaCross{
		candleSupplier: candleSupplier,
	}
}

func (ec *EmaCross) Evaluate() (*trade.Signal, bool) {
	candles := techan.NewTimeSeries()

	for _, currentCandle := range ec.candleSupplier.Candles() {
		candles.AddCandle(toTechanCandle(currentCandle))
	}

	price := techan.NewClosePriceIndicator(candles)
	priceEma := techan.NewEMAIndicator(price, 100)
	entryRule := newNearCrossUpIndicatorRule(priceEma, price)

	if entryRule.IsSatisfied(candles.LastIndex(), nil) {
		entryTarget := big.NewFloat(
			price.Calculate(candles.LastIndex()).Float(),
		)

		priceChangeFactor := 0.1 // TODO: use ATR indicator
		stopLossFactor := big.NewFloat(1 - priceChangeFactor)
		takeProfitFactor := big.NewFloat(1 + (2 * priceChangeFactor))

		stopLossTarget := new(big.Float).Mul(entryTarget, stopLossFactor)
		takeProfitTarget := new(big.Float).Mul(entryTarget, takeProfitFactor)

		return &trade.Signal{
			Type:             trade.LONG,
			EntryTarget:      entryTarget,
			TakeProfitTarget: takeProfitTarget,
			StopLossTarget:   stopLossTarget,
		}, true
	}

	return nil, false
}

func toTechanCandle(candle *candle.Candle) *techan.Candle {
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
