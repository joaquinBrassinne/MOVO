package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

type stubServices struct {
	services  []domain.Service
	schedules map[int64][]domain.Slot
}

func (s stubServices) Create(context.Context, domain.Service) (domain.Service, error) {
	panic("not used by the query")
}

func (s stubServices) List(_ context.Context, businessID int64) ([]domain.Service, error) {
	var out []domain.Service
	for _, svc := range s.services {
		if svc.BusinessID == businessID {
			out = append(out, svc)
		}
	}
	return out, nil
}

func (s stubServices) Find(context.Context, int64, int64) (domain.Service, error) {
	panic("not used by the query")
}

func (s stubServices) Save(context.Context, domain.Service) error { panic("not used by the query") }

func (s stubServices) Delete(context.Context, int64, int64) error { panic("not used by the query") }

func (s stubServices) Schedule(_ context.Context, _, serviceID int64) ([]domain.Slot, error) {
	return s.schedules[serviceID], nil
}

func (s stubServices) ReplaceSchedule(context.Context, int64, int64, []domain.Slot) error {
	panic("not used by the query")
}

func TestListMyServicesReturnsOnlyTheOwnersServicesWithTheirSchedule(t *testing.T) {
	t.Parallel()

	businesses := stubBusinesses{byOwner: map[int64]domain.Business{
		7: {ID: 1, OwnerID: 7},
		8: {ID: 2, OwnerID: 8},
	}}
	services := stubServices{
		services: []domain.Service{
			{ID: 10, BusinessID: 1, Name: "Yoga", Active: true},
			{ID: 11, BusinessID: 1, Name: "Old class"},
			{ID: 12, BusinessID: 2, Name: "Foreign", Active: true},
		},
		schedules: map[int64][]domain.Slot{10: {{Weekday: 1, Start: 540, End: 600}}},
	}
	query := queries.NewListMyServices(businesses, services)

	views, err := query.Handle(t.Context(), 7)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(views) != 2 || views[0].Service.Name != "Yoga" || len(views[0].Schedule) != 1 || views[1].Service.Active {
		t.Fatalf("views = %+v, want my two services, the inactive one included", views)
	}

	if _, err := query.Handle(t.Context(), 99); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("owner without business error = %v, want ErrBusinessNotFound", err)
	}

	empty, err := queries.NewListMyServices(businesses, stubServices{}).Handle(t.Context(), 8)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("no services = %#v, %v; want a non-nil empty list", empty, err)
	}
}
