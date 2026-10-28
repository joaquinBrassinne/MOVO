package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakeActivity struct {
	got      benefits.Activity
	err      error
	business int64
	since    *time.Time
	limit    int
}

func (f *fakeActivity) RecentActivity(_ context.Context, businessID int64, since *time.Time, limit int) (benefits.Activity, error) {
	f.business, f.since, f.limit = businessID, since, limit
	return f.got, f.err
}

func TestRecentActivityReadsTheBusinessOfTheSession(t *testing.T) {
	t.Parallel()

	mark := now.Add(-time.Minute)
	reader := &fakeActivity{got: benefits.Activity{NewCount: 4}}
	got, err := queries.NewRecentActivity(fakeOwners{business: 9}, reader).Handle(t.Context(), 7, &mark)
	if err != nil || got.NewCount != 4 || reader.business != 9 || reader.since != &mark || reader.limit != queries.ActivityLimit {
		t.Errorf("got %+v, %v; read business %d since %v limit %d", got, err, reader.business, reader.since, reader.limit)
	}
}

func TestRecentActivityNeedsABusiness(t *testing.T) {
	t.Parallel()

	if _, err := queries.NewRecentActivity(fakeOwners{err: benefits.ErrBusinessNotFound}, &fakeActivity{}).Handle(t.Context(), 7, nil); !errors.Is(err, benefits.ErrBusinessNotFound) {
		t.Errorf("error = %v, want ErrBusinessNotFound", err)
	}
}
