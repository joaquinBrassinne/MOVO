package postgres_test

import (
	"math"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
)

// reviewedRedemption inserts a coupon redeemed at redeemedAt and, when rating is not zero, a
// review of it with the status and creation time given.
func (f *conversionFixture) reviewedRedemption(t *testing.T, redeemedAt time.Time, rating int, status string, reviewedAt time.Time) {
	t.Helper()
	f.n++
	code := raceCode(f.n)
	hash := make([]byte, 32)
	copy(hash, code)
	var benefit, coupon int64
	if err := f.pool.QueryRow(t.Context(), `SELECT id FROM benefits WHERE business_id = $1`, f.business).Scan(&benefit); err != nil {
		t.Fatal(err)
	}
	created := redeemedAt.Add(-time.Hour)
	if err := f.pool.QueryRow(t.Context(), `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ($1, $2, $3, $4, $5, 'redeemed', $6, $7, $8, $9, $10) RETURNING id`,
		code, benefit, f.business, f.customer, hash, created.Add(48*time.Hour), created, redeemedAt, f.owner, f.branch).Scan(&coupon); err != nil {
		t.Fatal(err)
	}
	if rating == 0 {
		return
	}
	hiddenAt := any(nil)
	if status == "hidden" {
		hiddenAt = reviewedAt
	}
	if _, err := f.pool.Exec(t.Context(), `
INSERT INTO reviews (branch_id, business_id, benefit_code_id, customer_id, rating, display_name, status, hidden_at, created_at)
VALUES ($1, $2, $3, $4, $5, 'Ana', $6, $7, $8)`,
		f.branch, f.business, coupon, f.customer, rating, status, hiddenAt, reviewedAt); err != nil {
		t.Fatal(err)
	}
}

func TestReviewsOverTimeAveragesPublicReviewsAndCountsTheRedemptionsThatLeftOne(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	at := func(d, h, m, s int) time.Time { return time.Date(2028, 3, d, h, m, s, 0, argentina) }

	f.reviewedRedemption(t, at(12, 10, 0, 0), 5, "visible", at(12, 11, 0, 0))  // A
	f.reviewedRedemption(t, at(13, 10, 0, 0), 4, "reported", at(13, 11, 0, 0)) // B: reported still counts
	f.reviewedRedemption(t, at(13, 12, 0, 0), 1, "hidden", at(13, 13, 0, 0))   // C: hidden counts for nothing
	f.reviewedRedemption(t, at(14, 10, 0, 0), 0, "", time.Time{})              // D: no review
	f.reviewedRedemption(t, at(9, 0, 0, 0), 2, "visible", at(9, 0, 0, 0))      // F: first instant of the period
	f.reviewedRedemption(t, at(5, 10, 0, 0), 3, "visible", at(6, 10, 0, 0))    // E: previous period
	f.reviewedRedemption(t, at(8, 23, 0, 0), 4, "visible", at(8, 23, 59, 59))  // G: last second of the previous period
	f.reviewedRedemption(t, at(16, 0, 0, 0), 5, "visible", at(16, 0, 0, 1))    // after the period

	got, err := postgres.NewCoupons(f.pool).ReviewsOverTime(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	cur, prev := got.Current, got.Previous
	if cur.Reviews != 3 || cur.Average == nil || math.Abs(*cur.Average-11.0/3) > 1e-9 {
		t.Errorf("current reviews = %d with average %v, want 3 averaging 5, 4 and 2", cur.Reviews, cur.Average)
	}
	if cur.Redeemed != 5 || cur.WithReview != 3 {
		t.Errorf("current redeemed = %d with review %d, want 5 and 3 (the hidden one and the missing one do not)", cur.Redeemed, cur.WithReview)
	}
	if prev.Reviews != 2 || prev.Average == nil || *prev.Average != 3.5 || prev.Redeemed != 2 || prev.WithReview != 2 {
		t.Errorf("previous = %+v, want 2 reviews averaging 3.5 from 2 redemptions", prev)
	}
}

func TestReviewsOverTimeWithNoReviewsHasNoAverageAndIgnoresOtherBusinesses(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	when := time.Date(2028, 3, 12, 10, 0, 0, 0, argentina)

	f.reviewedRedemption(t, when, 0, "", time.Time{})
	other := newConversionFixtureIn(t, f, "other")
	other.reviewedRedemption(t, when, 5, "visible", when.Add(time.Hour))

	got, err := postgres.NewCoupons(f.pool).ReviewsOverTime(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current.Reviews != 0 || got.Current.Average != nil || got.Current.Redeemed != 1 || got.Current.WithReview != 0 {
		t.Errorf("current = %+v, want one redemption, no review and no average", got.Current)
	}
}
