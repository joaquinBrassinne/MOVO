package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

type fakePurger struct {
	got time.Time
	n   int64
	err error
}

func (f *fakePurger) PurgeHidden(_ context.Context, hiddenBefore time.Time) (int64, error) {
	f.got = hiddenBefore
	return f.n, f.err
}

func TestPurgeHiddenReviewsKeepsTheRetentionAndReportsTheCount(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	purger := &fakePurger{n: 3}

	n, err := commands.NewPurgeHiddenReviews(purger, func() time.Time { return now }).Handle(t.Context())
	if err != nil || n != 3 {
		t.Fatalf("Handle = %d, %v; want 3", n, err)
	}
	if want := now.Add(-72 * time.Hour); !purger.got.Equal(want) || domain.HiddenRetention != 72*time.Hour {
		t.Errorf("purged hidden before %v, want %v: 72 hours of grace", purger.got, want)
	}
}

func TestPurgeHiddenReviewsWrapsTheStoreError(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	_, err := commands.NewPurgeHiddenReviews(&fakePurger{err: boom}, nil).Handle(t.Context())
	if !errors.Is(err, boom) {
		t.Errorf("Handle error = %v, want it to wrap the store error", err)
	}
}
