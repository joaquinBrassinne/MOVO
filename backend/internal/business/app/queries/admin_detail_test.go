package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

type detailDirectory struct {
	rows    map[int64]business.AdminRow
	history map[int64][]business.AuditRecord
}

func (d detailDirectory) List(context.Context, business.AdminFilter) (business.AdminPage, error) {
	panic("not used by the query")
}

func (d detailDirectory) Find(_ context.Context, id int64) (business.AdminRow, error) {
	row, ok := d.rows[id]
	if !ok {
		return business.AdminRow{}, business.ErrBusinessNotFound
	}
	return row, nil
}

func (d detailDirectory) History(_ context.Context, id int64) ([]business.AuditRecord, error) {
	return d.history[id], nil
}

func TestGetBusinessDetailGathersTheRowBranchesServicesAndHistory(t *testing.T) {
	t.Parallel()

	directory := detailDirectory{
		rows:    map[int64]business.AdminRow{1: {Business: domain.Business{ID: 1, Name: "Mine"}, OwnerEmail: "o@example.com"}},
		history: map[int64][]business.AuditRecord{1: {{Action: "business.verify", ActorEmail: "root@example.com"}}},
	}
	branches := stubBranches{branches: []domain.Branch{{ID: 10, BusinessID: 1, Name: "Centro"}, {ID: 11, BusinessID: 2, Name: "Foreign"}}}
	services := stubServices{services: []domain.Service{{ID: 5, BusinessID: 1, Name: "Yoga", Active: true}, {ID: 6, BusinessID: 2, Name: "Foreign"}}}

	got, err := queries.NewGetBusinessDetail(directory, branches, services).Handle(t.Context(), 1)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Row.OwnerEmail != "o@example.com" || len(got.Branches) != 1 || got.Branches[0].ID != 10 ||
		len(got.Services) != 1 || got.Services[0].ID != 5 || len(got.History) != 1 || got.History[0].Action != "business.verify" {
		t.Errorf("detail = %+v, want only the data of business 1", got)
	}

	if _, err := queries.NewGetBusinessDetail(directory, branches, services).Handle(t.Context(), 99); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("missing business error = %v, want ErrBusinessNotFound", err)
	}
}
