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

type fakeOwners struct {
	business int64
	err      error
}

func (f fakeOwners) EditableBusinessOf(context.Context, int64) (int64, error) {
	return f.business, f.err
}

// fakeReader answers a fixed preview and records what it was asked.
type fakeReader struct {
	preview benefits.CouponPreview
	err     error
	asked   []string
}

func (f *fakeReader) Preview(_ context.Context, businessID int64, code string) (benefits.CouponPreview, error) {
	f.asked = append(f.asked, code)
	_ = businessID
	return f.preview, f.err
}

var now = time.Date(2028, 3, 15, 12, 0, 0, 0, time.UTC)

func clock() time.Time { return now }

func pending(expires time.Time) benefits.CouponPreview {
	return benefits.CouponPreview{
		Coupon:       domain.Coupon{Code: "ABCDEFGH", Status: domain.StatusPending, ExpiresAt: expires},
		Benefit:      domain.Benefit{ID: 1, Type: domain.TypePercent, Percent: 15},
		CustomerName: "Ana",
	}
}

func TestPreviewShowsTheStatusAtTheMomentOfTheLook(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		preview benefits.CouponPreview
		want    domain.Status
	}{
		"pending in time":     {pending(now.Add(time.Hour)), domain.StatusPending},
		"pending but overdue": {pending(now.Add(-time.Second)), domain.StatusExpired},
		"redeemed stays": {func() benefits.CouponPreview {
			p := pending(now.Add(-time.Hour))
			p.Coupon.Status = domain.StatusRedeemed
			return p
		}(), domain.StatusRedeemed},
	} {
		got, err := queries.NewPreviewCode(fakeOwners{business: 1}, &fakeReader{preview: tc.preview}, clock).Handle(t.Context(), 7, "abcd-efgh")
		if err != nil || got.Status != tc.want || got.Code != "ABCDEFGH" || got.CustomerName != "Ana" || got.Benefit.Percent != 15 {
			t.Errorf("%s: view = %+v, %v; want status %q", name, got, err, tc.want)
		}
	}
}

func TestPreviewLooksUpTheNormalizedCodeAndRefusesMalformedOnesWithoutReading(t *testing.T) {
	t.Parallel()

	reader := &fakeReader{preview: pending(now.Add(time.Hour))}
	query := queries.NewPreviewCode(fakeOwners{business: 1}, reader, clock)

	if _, err := query.Handle(t.Context(), 7, " abcd efgh "); err != nil || len(reader.asked) != 1 || reader.asked[0] != "ABCDEFGH" {
		t.Fatalf("asked %v, %v; want one lookup of ABCDEFGH", reader.asked, err)
	}
	if _, err := query.Handle(t.Context(), 7, "nope"); !errors.Is(err, benefits.ErrCouponNotFound) || len(reader.asked) != 1 {
		t.Errorf("malformed: error = %v after %d lookups, want ErrCouponNotFound and no new lookup", err, len(reader.asked))
	}
}

func TestPreviewNeedsABusinessAndPassesNotFoundThrough(t *testing.T) {
	t.Parallel()

	if _, err := queries.NewPreviewCode(fakeOwners{err: benefits.ErrBusinessNotFound}, &fakeReader{}, clock).Handle(t.Context(), 7, "ABCDEFGH"); !errors.Is(err, benefits.ErrBusinessNotFound) {
		t.Errorf("no business: error = %v", err)
	}
	if _, err := queries.NewPreviewCode(fakeOwners{business: 1}, &fakeReader{err: benefits.ErrCouponNotFound}, clock).Handle(t.Context(), 7, "ABCDEFGH"); !errors.Is(err, benefits.ErrCouponNotFound) {
		t.Errorf("unknown coupon: error = %v", err)
	}
}
