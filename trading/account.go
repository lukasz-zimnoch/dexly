package trading

import "math/big"

type AccountRepository interface {
	CreateAccount(account *Account) error

	Account(accountID ID) (*Account, error)
}

type Account struct {
	ID    ID
	Email string

	Exchange          string
	ExchangeApiKey    string
	ExchangeSecretKey string // TODO: Store credentials in a secure way.

	RiskFactor         *big.Float
	OpenPositionsLimit int
}

type AccountWalletItem struct {
	*Account

	Asset           Asset
	Balance         *big.Float
	TakerCommission *big.Float
}
