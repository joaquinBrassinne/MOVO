package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/commands"
)

type fakeExpirer struct {
	at  time.Time
	n   int64
	err error
}

func (f *fakeExpirer) ExpireOverdue(_ context.Context, now time.Time) (int64, error) {
	f.at = now
	return f.n, f.err
}

func TestExpireCouponsSweepsAtTheClockOfTheApp(t *testing.T) {
	t.Parallel()

	at := time.Date(2028, 3, 15, 3, 0, 0, 0, time.UTC)
	fake := &fakeExpirer{n: 3}
	n, err := commands.NewExpireCoupons(fake, func() time.Time { return at }).Handle(t.Context())
	if err != nil || n != 3 || !fake.at.Equal(at) {
		t.Errorf("sweep = %d, %v at %v; want 3 at %v", n, err, fake.at, at)
	}

	boom := errors.New("boom")
	if _, err := commands.NewExpireCoupons(&fakeExpirer{err: boom}, nil).Handle(t.Context()); !errors.Is(err, boom) {
		t.Errorf("error = %v, want the store error wrapped", err)
	}
}
