package postgres

import (
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
	// TODO: Implementation.
	panic("not implemented")
}

func (wr *WorkloadRepository) Workloads() ([]*trading.Workload, error) {
	// TODO: Implementation.
	panic("not implemented")
}
