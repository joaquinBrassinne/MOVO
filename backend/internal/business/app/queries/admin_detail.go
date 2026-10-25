package queries

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// BusinessDetail is everything the super admin sees about one business: the list row,
// its branches and services, and the history of what admins did to it.
type BusinessDetail struct {
	Row      business.AdminRow
	Branches []domain.Branch
	Services []domain.Service
	History  []business.AuditRecord
}

// GetBusinessDetail reads one business for the super admin.
type GetBusinessDetail struct {
	directory business.AdminDirectory
	branches  business.BranchRepository
	services  business.ServiceRepository
}

// NewGetBusinessDetail wires the GetBusinessDetail query.
func NewGetBusinessDetail(directory business.AdminDirectory, branches business.BranchRepository, services business.ServiceRepository) *GetBusinessDetail {
	return &GetBusinessDetail{directory: directory, branches: branches, services: services}
}

// Handle returns business.ErrBusinessNotFound when there is no such business. The caller
// must already have checked the super_admin role.
func (q *GetBusinessDetail) Handle(ctx context.Context, id int64) (BusinessDetail, error) {
	row, err := q.directory.Find(ctx, id)
	if err != nil {
		return BusinessDetail{}, fmt.Errorf("find business: %w", err)
	}
	branches, err := q.branches.List(ctx, id)
	if err != nil {
		return BusinessDetail{}, fmt.Errorf("list branches: %w", err)
	}
	services, err := q.services.List(ctx, id)
	if err != nil {
		return BusinessDetail{}, fmt.Errorf("list services: %w", err)
	}
	history, err := q.directory.History(ctx, id)
	if err != nil {
		return BusinessDetail{}, fmt.Errorf("read history: %w", err)
	}

	return BusinessDetail{Row: row, Branches: branches, Services: services, History: history}, nil
}
