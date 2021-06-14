package trading

import "math/big"

type Asset string

type PairSymbol string

type Pair struct {
	Base, Quote Asset
}

func (p Pair) Symbol() PairSymbol {
	return PairSymbol(p.Base + p.Quote)
}

type Balances map[Asset]*big.Float

func (bm Balances) BalanceOf(asset Asset) *big.Float {
	for balanceAsset, balanceValue := range bm {
		if balanceAsset == asset {
			return balanceValue
		}
	}

	return big.NewFloat(0)
}
