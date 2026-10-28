package queries_test

import (
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

func TestViewMyCouponLooksUpTheHashOfTheTokenAndShowsTheStatusAtTheMomentOfTheLook(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		expires time.Time
		want    domain.Status
	}{
		"pending in time":     {now.Add(time.Hour), domain.StatusPending},
		"pending but overdue": {now.Add(-time.Second), domain.StatusExpired},
	} {
		reader := &fakeReader{held: benefits.HeldCoupon{CouponPreview: pending(tc.expires), BusinessName: "Studio"}}
		got, err := queries.NewViewMyCoupon(reader, clock).Handle(t.Context(), "the-token")
		if err != nil || got.Status != tc.want || got.Code != "ABCDEFGH" || got.BusinessName != "Studio" || got.Benefit.Percent != 15 {
			t.Errorf("%s: view = %+v, %v; want status %q", name, got, err, tc.want)
		}
		if string(reader.hash) != string(domain.HashAccessToken("the-token")) {
			t.Errorf("%s: looked up %x, want the SHA-256 of the token, never the token", name, reader.hash)
		}
	}
}

func TestViewMyCouponPassesNotFoundThrough(t *testing.T) {
	t.Parallel()

	_, err := queries.NewViewMyCoupon(&fakeReader{err: benefits.ErrCouponNotFound}, clock).Handle(t.Context(), "nope")
	if !errors.Is(err, benefits.ErrCouponNotFound) {
		t.Errorf("error = %v, want ErrCouponNotFound", err)
	}
}
