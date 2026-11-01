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

type fakeProjectionQuotas struct {
	quota domain.Quota
	asked int
	start time.Time
}

func (f *fakeProjectionQuotas) Quota(_ context.Context, _ int64, start, _ time.Time) (domain.Quota, error) {
	f.asked++
	f.start = start
	return f.quota, nil
}

func (f *fakeProjectionQuotas) SetQuotaLimit(context.Context, int64, *int, time.Time, time.Time) (domain.Quota, error) {
	return domain.Quota{}, nil
}

func TestQuotaProjectionReadsTheMonthInCourseAndSaysWhenItResets(t *testing.T) {
	t.Parallel()

	limit := 100
	quotas := &fakeProjectionQuotas{quota: domain.Quota{Limit: &limit, Generated: 100}}
	got, err := queries.NewGetQuotaProjection(proGate(), quotas, clock).Handle(t.Context(), 7)
	if err != nil {
		t.Fatal(err)
	}
	// "now" is March 15th 2028 in Argentina.
	if got.Status != domain.ProjectionReached || got.ResetsOn != "2028-04-01" || quotas.start.Day() != 1 || quotas.start.Month() != 3 {
		t.Errorf("projection = %+v (month from %v), want exhausted, resetting on 2028-04-01, of March", got, quotas.start)
	}
}

func TestQuotaProjectionIsRefusedBeforeTheQuotaIsReadForAStarter(t *testing.T) {
	t.Parallel()

	quotas := &fakeProjectionQuotas{}
	denied := queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: false})
	if _, err := queries.NewGetQuotaProjection(denied, quotas, clock).Handle(t.Context(), 7); !errors.Is(err, benefits.ErrPlanUpgradeRequired) || quotas.asked != 0 {
		t.Errorf("Starter: err = %v, reads = %d; want ErrPlanUpgradeRequired and no read", err, quotas.asked)
	}
}
