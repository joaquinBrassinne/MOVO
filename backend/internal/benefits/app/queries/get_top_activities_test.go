package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakeDemand struct {
	rows   []benefits.ActivityDemand
	asked  []int64
	limit  int
	period benefits.MetricsPeriod
}

func (f *fakeDemand) TopActivities(_ context.Context, businessID int64, p benefits.MetricsPeriod, limit int) ([]benefits.ActivityDemand, error) {
	f.asked = append(f.asked, businessID)
	f.period, f.limit = p, limit
	return f.rows, nil
}

func TestTopActivitiesReadTheBusinessOfTheSessionAndAskForFive(t *testing.T) {
	t.Parallel()

	reader := &fakeDemand{rows: []benefits.ActivityDemand{{ServiceID: 1, Name: "Yoga", Views: 4, Clicks: 2}}}
	got, err := queries.NewGetTopActivities(fakeOwners{business: 42}, reader, clock).Handle(t.Context(), 7, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.asked) != 1 || reader.asked[0] != 42 || reader.limit != queries.TopActivitiesLimit {
		t.Errorf("asked %v with limit %d, want business 42 and limit %d", reader.asked, reader.limit, queries.TopActivitiesLimit)
	}
	if len(got.Activities) != 1 || got.Activities[0].Name != "Yoga" || got.Period.Days != 7 {
		t.Errorf("result = %+v, want the reader's ranking over 7 days", got)
	}
}

func TestTopActivitiesAnswerAnEmptyListNotNil(t *testing.T) {
	t.Parallel()

	got, err := queries.NewGetTopActivities(fakeOwners{business: 1}, &fakeDemand{}, clock).Handle(t.Context(), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Activities == nil || len(got.Activities) != 0 || got.Period.Days != queries.DefaultMetricsDays {
		t.Errorf("result = %+v, want an empty non-nil list over the default period", got)
	}
}

func TestTopActivitiesRefuseOtherPeriodsAndMissingBusinesses(t *testing.T) {
	t.Parallel()

	reader := &fakeDemand{}
	if _, err := queries.NewGetTopActivities(fakeOwners{business: 1}, reader, clock).Handle(t.Context(), 1, 14); !errors.Is(err, benefits.ErrMetricsPeriodInvalid) {
		t.Errorf("period 14: err = %v, want ErrMetricsPeriodInvalid", err)
	}
	if _, err := queries.NewGetTopActivities(fakeOwners{err: benefits.ErrBusinessNotFound}, reader, clock).Handle(t.Context(), 1, 30); !errors.Is(err, benefits.ErrBusinessNotFound) {
		t.Errorf("no business: err = %v, want ErrBusinessNotFound", err)
	}
	if len(reader.asked) != 0 {
		t.Errorf("the reader was asked %d times, want none", len(reader.asked))
	}
}
