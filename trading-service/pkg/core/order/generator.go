package order

import (
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/candle"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/core/logger"
	technicalbig "github.com/sdcoffey/big"
	technical "github.com/sdcoffey/techan"
	"math/big"
	"sync"
	"time"
)

type Side int

const (
	BUY Side = iota
	SELL
)

func (s Side) String() string {
	switch s {
	case BUY:
		return "BUY"
	case SELL:
		return "SELL"
	}

	return ""
}

type Order struct {
	Side         Side
	Price        *big.Float
	Amount       *big.Float
	CreationTime time.Time
}

func newBuyOrder(price, amount *big.Float) *Order {
	return &Order{
		Side:         BUY,
		Price:        price,
		Amount:       amount,
		CreationTime: time.Now(),
	}
}

func newSellOrder(price, amount *big.Float) *Order {
	return &Order{
		Side:         SELL,
		Price:        price,
		Amount:       amount,
		CreationTime: time.Now(),
	}
}

func (o *Order) String() string {
	return fmt.Sprintf(
		"side: %v, amount: %v, price: %v",
		o.Side,
		o.Amount.String(),
		o.Price.String(),
	)
}

type candleSource interface {
	Candles() []*candle.Candle
}

type accountManager interface {
	Balance() (*big.Float, error)
}

type Generator struct {
	logger logger.Logger

	recordMutex sync.Mutex
	record      *technical.TradingRecord

	candleSource   candleSource
	accountManager accountManager

	balanceRiskFactor *big.Float
	priceChangeFactor *big.Float
}

func NewGenerator(
	logger logger.Logger,
	candleSource candleSource,
	accountManager accountManager,
) *Generator {
	return &Generator{
		logger:            logger,
		record:            technical.NewTradingRecord(),
		candleSource:      candleSource,
		accountManager:    accountManager,
		balanceRiskFactor: big.NewFloat(0.02),
		priceChangeFactor: big.NewFloat(0.1),
	}
}

func (g Generator) GenerateOrder() (*Order, bool) {
	g.recordMutex.Lock()
	defer g.recordMutex.Unlock()

	series := technical.NewTimeSeries()

	for _, currentCandle := range g.candleSource.Candles() {
		series.AddCandle(newTechnicalCandle(currentCandle))
	}

	position := g.record.CurrentPosition()
	priceIndicator := technical.NewClosePriceIndicator(series)

	if position.IsNew() {
		priceEma := technical.NewEMAIndicator(priceIndicator, 100)
		entryRule := newNearCrossUpIndicatorRule(priceEma, priceIndicator)

		if entryRule.IsSatisfied(series.LastIndex(), g.record) {
			g.logger.Infof("detected entry rule fulfillment")

			balance, err := g.accountManager.Balance()
			if err != nil {
				g.logger.Errorf("could not get account balance: [%v]", err)
				return nil, false
			}

			price := asBigFloat(priceIndicator.Calculate(series.LastIndex()))
			priceChange := new(big.Float).Mul(price, g.priceChangeFactor)
			balanceAtRisk := new(big.Float).Mul(balance, g.balanceRiskFactor)
			amount := new(big.Float).Quo(balanceAtRisk, priceChange)

			maxAmount := new(big.Float).Quo(balance, price)
			if amount.Cmp(maxAmount) == 1 {
				amount = maxAmount
			}

			return newBuyOrder(price, amount), true
		}
	} else if position.IsOpen() {
		priceChangeFactor, _ := g.priceChangeFactor.Float64()
		stopLossFactor := 1 - priceChangeFactor
		takeProfitFactor := 1 + (2 * priceChangeFactor)

		entryPrice := position.EntranceOrder().Price.Float()
		stopLoss := technical.NewConstantIndicator(entryPrice * stopLossFactor)
		takeProfit := technical.NewConstantIndicator(entryPrice * takeProfitFactor)

		exitRule := technical.Or(
			technical.UnderIndicatorRule{
				First:  priceIndicator,
				Second: stopLoss,
			},
			technical.OverIndicatorRule{
				First:  priceIndicator,
				Second: takeProfit,
			},
		)

		if exitRule.IsSatisfied(series.LastIndex(), g.record) {
			g.logger.Infof("detected exit rule fulfillment")

			price := asBigFloat(priceIndicator.Calculate(series.LastIndex()))
			amount := asBigFloat(position.EntranceOrder().Amount)

			return newSellOrder(price, amount), true
		}
	}

	return nil, false
}

func (g *Generator) RecordOrderExecution(order *Order) {
	g.recordMutex.Lock()
	defer g.recordMutex.Unlock()

	g.record.Operate(newTechnicalOrder(order))
}

func newTechnicalCandle(candle *candle.Candle) *technical.Candle {
	period := technical.TimePeriod{
		Start: candle.OpenTime,
		End:   candle.CloseTime,
	}

	technicalCandle := technical.NewCandle(period)

	technicalCandle.OpenPrice = technicalbig.NewFromString(candle.OpenPrice)
	technicalCandle.ClosePrice = technicalbig.NewFromString(candle.ClosePrice)
	technicalCandle.MaxPrice = technicalbig.NewFromString(candle.MaxPrice)
	technicalCandle.MinPrice = technicalbig.NewFromString(candle.MinPrice)
	technicalCandle.Volume = technicalbig.NewFromString(candle.Volume)
	technicalCandle.TradeCount = candle.TradeCount

	return technicalCandle
}

func newTechnicalOrder(order *Order) technical.Order {
	return technical.Order{
		Side:          technical.OrderSide(order.Side),
		Security:      "",
		Price:         technicalbig.NewFromString(order.Price.String()),
		Amount:        technicalbig.NewFromString(order.Amount.String()),
		ExecutionTime: order.CreationTime,
	}
}

func asBigFloat(value technicalbig.Decimal) *big.Float {
	return big.NewFloat(value.Float())
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
	_ *technical.TradingRecord,
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
