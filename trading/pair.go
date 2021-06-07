package trading

import "strings"

type Asset string

type Pair struct {
	Base, Quote Asset
}

func ParsePair(pair string) Pair {
	symbols := strings.Split(pair, "/")

	return Pair{
		Base:  Asset(symbols[0]),
		Quote: Asset(symbols[1]),
	}
}

func (p Pair) String() string {
	return string(p.Base + p.Quote)
}
