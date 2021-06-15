package postgres

import (
	"fmt"
	"github.com/lukasz-zimnoch/dexly/trading"
)

type WorkloadRepository struct {
	client    *Client
	idService trading.IDService
}

func NewWorkloadRepository(
	client *Client,
	idService trading.IDService,
) *WorkloadRepository {
	return &WorkloadRepository{client, idService}
}

func (wr *WorkloadRepository) CreateWorkload(workload *trading.Workload) error {
	query := `INSERT INTO 
    	workload (id, account_id, base_asset, quote_asset) 
    	VALUES (:id, :account_id, :base_asset, :quote_asset)`

	workloadRow, err := new(workloadRow).wrap(workload)
	if err != nil {
		return fmt.Errorf(
			"could not convert workload [%v] to pg row: [%v]",
			workload.ID,
			err,
		)
	}

	_, err = wr.client.instance().NamedExec(query, workloadRow)
	if err != nil {
		return fmt.Errorf(
			"could not execute command for workload [%v]: [%v]",
			workload.ID,
			err,
		)
	}

	return nil
}

func (wr *WorkloadRepository) Workloads() ([]*trading.Workload, error) {
	var selectResult []struct {
		workloadRow `db:"workload"`
		accountRow  `db:"account"`
	}

	query :=
		`SELECT 
       		w.id "workload.id",
       		w.account_id "workload.account_id",
       		w.base_asset "workload.base_asset",
       		w.quote_asset "workload.quote_asset",
       		a.id "account.id",
       		a.email "account.email",
       		a.exchange "account.exchange",
       		a.exchange_api_key "account.exchange_api_key",
       		a.exchange_secret_key "account.exchange_secret_key",
       		a.risk_factor "account.risk_factor",
       		a.open_position_limit "account.open_position_limit"
		FROM workload w
		JOIN account a ON a.id = w.account_id`

	err := wr.client.instance().Select(
		&selectResult,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("could not execute query: [%v]", err)
	}

	workloads := make([]*trading.Workload, 0)

	for _, result := range selectResult {
		account, err := result.accountRow.unwrap(wr.idService)
		if err != nil {
			return nil, fmt.Errorf(
				"could not convert account [%v] from pg row: [%v]",
				result.accountRow.ID,
				err,
			)
		}

		workload, err := result.workloadRow.unwrap(wr.idService)
		if err != nil {
			return nil, fmt.Errorf(
				"could not convert workload [%v] from pg row: [%v]",
				result.workloadRow.ID,
				err,
			)
		}

		workload.Account = account
		workloads = append(workloads, workload)
	}

	return workloads, nil
}

type workloadRow struct {
	ID         string
	AccountID  string `db:"account_id"`
	BaseAsset  string `db:"base_asset"`
	QuoteAsset string `db:"quote_asset"`
}

func (wr *workloadRow) wrap(workload *trading.Workload) (*workloadRow, error) {
	wr.ID = workload.ID.String()
	wr.AccountID = workload.Account.ID.String()
	wr.BaseAsset = string(workload.Pair.Base)
	wr.QuoteAsset = string(workload.Pair.Quote)

	return wr, nil
}

func (wr *workloadRow) unwrap(
	idService trading.IDService,
) (*trading.Workload, error) {
	ID, err := idService.NewIDFromString(wr.ID)
	if err != nil {
		return nil, err
	}

	pair := trading.Pair{
		Base:  trading.Asset(wr.BaseAsset),
		Quote: trading.Asset(wr.QuoteAsset),
	}

	return &trading.Workload{
		ID:      ID,
		Account: nil, // Account should be set outside.
		Pair:    pair,
	}, nil
}
