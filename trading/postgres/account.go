package postgres

import (
	"fmt"
	"github.com/jackc/pgtype"
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
	query := `INSERT INTO 
    	account (id, email, exchange, exchange_api_key, exchange_secret_key, 
    	         risk_factor, open_position_limit) 
    	VALUES (:id, :email, :exchange, :exchange_api_key, :exchange_secret_key, 
    	        :risk_factor, :open_position_limit)`

	accountRow, err := new(accountRow).wrap(account)
	if err != nil {
		return fmt.Errorf(
			"could not convert account [%v] to pg row: [%v]",
			account.ID,
			err,
		)
	}

	_, err = ar.client.instance().NamedExec(query, accountRow)
	if err != nil {
		return fmt.Errorf(
			"could not execute command for account [%v]: [%v]",
			account.ID,
			err,
		)
	}

	return nil
}

func (ar *AccountRepository) Account(
	accountID trading.ID,
) (*trading.Account, error) {
	var accountRow accountRow

	query := `SELECT * FROM account WHERE id = $1`

	err := ar.client.instance().Get(
		&accountRow,
		query,
		accountID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not execute query: [%v]", err)
	}

	return accountRow.unwrap(ar.idService)
}

type accountRow struct {
	ID                string
	Email             string
	Exchange          string
	ExchangeApiKey    string         `db:"exchange_api_key"`
	ExchangeSecretKey string         `db:"exchange_secret_key"`
	RiskFactor        pgtype.Numeric `db:"risk_factor"`
	OpenPositionLimit int            `db:"open_position_limit"`
}

func (ar *accountRow) wrap(account *trading.Account) (*accountRow, error) {
	riskFactor, err := floatToNumeric(account.RiskFactor)
	if err != nil {
		return nil, err
	}

	ar.ID = account.ID.String()
	ar.Email = account.Email
	ar.Exchange = account.Exchange
	ar.ExchangeApiKey = account.ExchangeApiKey
	ar.ExchangeSecretKey = account.ExchangeSecretKey
	ar.RiskFactor = riskFactor
	ar.OpenPositionLimit = account.OpenPositionsLimit

	return ar, nil
}

func (ar *accountRow) unwrap(
	idService trading.IDService,
) (*trading.Account, error) {
	ID, err := idService.NewIDFromString(ar.ID)
	if err != nil {
		return nil, err
	}

	riskFactor, err := numericToFloat(ar.RiskFactor)
	if err != nil {
		return nil, err
	}

	return &trading.Account{
		ID:                 ID,
		Email:              ar.Email,
		Exchange:           ar.Exchange,
		ExchangeApiKey:     ar.ExchangeApiKey,
		ExchangeSecretKey:  ar.ExchangeSecretKey,
		RiskFactor:         riskFactor,
		OpenPositionsLimit: ar.OpenPositionLimit,
	}, nil
}
