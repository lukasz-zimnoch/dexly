package postgres

import (
	"github.com/lukasz-zimnoch/dexly/trading"
)

type AccountRepository struct {
	client    *Client
	idService trading.IDService
}

func NewAccountRepository(
	client *Client,
	idService trading.IDService,
) *AccountRepository {
	return &AccountRepository{client, idService}
}

func (ar *AccountRepository) CreateAccount(account *trading.Account) error {
	// TODO: Implementation.
	panic("not implemented")
}

func (ar *AccountRepository) Account(
	accountID trading.ID,
) (*trading.Account, error) {
	// TODO: Implementation.
	panic("not implemented")
}
