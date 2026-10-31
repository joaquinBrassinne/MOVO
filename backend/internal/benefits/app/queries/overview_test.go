package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

type fakeOverview struct {
	got      benefits.Overview
	business int64
	at       time.Time
}

func (f *fakeOverview) Overview(_ context.Context, businessID int64, now time.Time) (benefits.Overview, error) {
	f.business, f.at = businessID, now
	return f.got, nil
}

type fakeUsage struct {
	usage map[int64]benefits.Usage
	start time.Time
}

func (f *fakeUsage) MonthlyUsage(_ context.Context, _ int64, start, _ time.Time) (map[int64]benefits.Usage, error) {
	f.start = start
	return f.usage, nil
}

type fakeBenefits struct{ list []domain.Benefit }

func (f fakeBenefits) Create(context.Context, domain.Benefit) (domain.Benefit, error) {
	return domain.Benefit{}, nil
}
func (f fakeBenefits) List(context.Context, int64) ([]domain.Benefit, error) { return f.list, nil }
func (f fakeBenefits) Find(context.Context, int64, int64) (domain.Benefit, error) {
	return domain.Benefit{}, nil
}
func (f fakeBenefits) Save(context.Context, domain.Benefit) error { return nil }
func (f fakeBenefits) SetActive(context.Context, int64, int64, bool) (domain.Benefit, error) {
	return domain.Benefit{}, nil
}

func TestOverviewReadsTheBusinessOfTheSession(t *testing.T) {
	t.Parallel()

	reader := &fakeOverview{got: benefits.Overview{ActiveCoupons: 12}}
	got, err := queries.NewGetOverview(fakeOwners{business: 9}, reader, clock).Handle(t.Context(), 7)
	if err != nil || got.ActiveCoupons != 12 || reader.business != 9 || !reader.at.Equal(now) {
		t.Errorf("got %+v, %v; read business %d at %v", got, err, reader.business, reader.at)
	}
	if _, err := queries.NewGetOverview(fakeOwners{err: benefits.ErrBusinessNotFound}, reader, clock).Handle(t.Context(), 7); !errors.Is(err, benefits.ErrBusinessNotFound) {
		t.Errorf("no business: %v, want ErrBusinessNotFound", err)
	}
}

func TestListMyBenefitsAddsTheUsageOfTheMonthAndZerosForTheRest(t *testing.T) {
	t.Parallel()

	usage := &fakeUsage{usage: map[int64]benefits.Usage{2: {Generated: 19, Redeemed: 16}}}
	repo := fakeBenefits{list: []domain.Benefit{{ID: 1}, {ID: 2}}}
	got, err := queries.NewListMyBenefits(fakeOwners{business: 9}, repo, usage, clock).Handle(t.Context(), 7)
	if err != nil || len(got) != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if got[0].Generated != 0 || got[0].Redeemed != 0 || got[1].Generated != 19 || got[1].Redeemed != 16 {
		t.Errorf("usage = %+v, want zeros for the first and 19/16 for the second", got)
	}
	if start, _ := domain.QuotaMonth(now); !usage.start.Equal(start) {
		t.Errorf("read the month starting %v, want %v", usage.start, start)
	}
}
