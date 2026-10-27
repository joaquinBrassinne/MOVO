package advisory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/advisory"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

func TestRunExclusiveSkipsWhileAnotherHolderRuns(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()

	inside := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := advisory.RunExclusive(ctx, pool, 42, func(context.Context) error {
			close(inside)
			<-release
			return nil
		})
		done <- err
	}()
	<-inside

	ran, err := advisory.RunExclusive(ctx, pool, 42, func(context.Context) error {
		t.Error("the second holder ran while the first still held the lock")
		return nil
	})
	if err != nil || ran {
		t.Errorf("second holder = ran %v, %v; want it skipped without error", ran, err)
	}
	// Another key is independent.
	if ran, err := advisory.RunExclusive(ctx, pool, 43, func(context.Context) error { return nil }); err != nil || !ran {
		t.Errorf("other key = ran %v, %v; want it to run", ran, err)
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("first holder: %v", err)
	}
	// Once released, the lock can be taken again, and a failing job reports its error.
	boom := errors.New("boom")
	ran, err = advisory.RunExclusive(ctx, pool, 42, func(context.Context) error { return boom })
	if !ran || !errors.Is(err, boom) {
		t.Errorf("after release = ran %v, %v; want it to run and return the job error", ran, err)
	}
	if ran, err := advisory.RunExclusive(ctx, pool, 42, func(context.Context) error { return nil }); err != nil || !ran {
		t.Errorf("after a failing job = ran %v, %v; want the lock released", ran, err)
	}
}
