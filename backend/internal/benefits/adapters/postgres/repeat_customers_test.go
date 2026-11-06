package postgres_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
)

// customer inserts another customer and returns the id.
func (f *conversionFixture) newCustomer(t *testing.T, phone string) int64 {
	t.Helper()
	var id int64
	if err := f.pool.QueryRow(t.Context(), `INSERT INTO customers (name, phone) VALUES ('C ' || $1::text, $1) RETURNING id`, phone).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// redeemedBy inserts a coupon of a customer, redeemed at the instant given.
func (f *conversionFixture) redeemedBy(t *testing.T, customer int64, redeemedAt time.Time) {
	t.Helper()
	f.n++
	code := raceCode(f.n)
	hash := make([]byte, 32)
	copy(hash, code)
	var benefit int64
	if err := f.pool.QueryRow(t.Context(), `SELECT id FROM benefits WHERE business_id = $1`, f.business).Scan(&benefit); err != nil {
		t.Fatal(err)
	}
	created := redeemedAt.Add(-time.Hour)
	if _, err := f.pool.Exec(t.Context(), `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ($1, $2, $3, $4, $5, 'redeemed', $6, $7, $8, $9, $10)`,
		code, benefit, f.business, customer, hash, created.Add(48*time.Hour), created, redeemedAt, f.owner, f.branch); err != nil {
		t.Fatal(err)
	}
}

func TestRepeatCustomersCountsPeopleOnceWhateverTheirNumberOfRedemptions(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	ana, beto, cata, dani := f.customer, f.newCustomer(t, "5491100000002"), f.newCustomer(t, "5491100000003"), f.newCustomer(t, "5491100000004")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	at := func(d, h int) time.Time { return time.Date(2028, 3, d, h, 0, 0, 0, argentina) }

	f.redeemedBy(t, ana, at(10, 10)) // Ana: three times in the period
	f.redeemedBy(t, ana, at(11, 10))
	f.redeemedBy(t, ana, at(14, 10))
	f.redeemedBy(t, beto, at(12, 10)) // Beto: twice
	f.redeemedBy(t, beto, at(13, 10))
	f.redeemedBy(t, cata, at(12, 10)) // Cata: once
	f.redeemedBy(t, dani, at(5, 10))  // Dani: once, in the previous period
	f.redeemedBy(t, cata, at(6, 10))  // Cata also came in the previous period, once

	got, err := postgres.NewCoupons(f.pool).RepeatCustomers(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	if want := (benefits.CustomerCounts{OneRedemption: 1, Repeat: 2}); got.Current != want {
		t.Errorf("current = %+v, want %+v", got.Current, want)
	}
	if want := (benefits.CustomerCounts{OneRedemption: 2}); got.Previous != want {
		t.Errorf("previous = %+v, want %+v", got.Previous, want)
	}
}

func TestRepeatCustomersIgnoresOtherBusinessesAndUnredeemedCoupons(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	when := time.Date(2028, 3, 12, 10, 0, 0, 0, argentina)

	f.redeemedBy(t, f.customer, when)
	f.coupon(t, when, false, f.branch) // generated, never redeemed: not a return
	f.coupon(t, when, false, f.branch)
	other := newConversionFixtureIn(t, f, "other")
	other.redeemedBy(t, f.customer, when) // the same person at another business is not a return here
	other.redeemedBy(t, f.customer, when.Add(time.Hour))

	got, err := postgres.NewCoupons(f.pool).RepeatCustomers(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != (benefits.CustomerCounts{OneRedemption: 1}) || got.CouponsGenerated != 3 {
		t.Errorf("got %+v with %d coupons generated, want one customer once and 3 coupons", got.Current, got.CouponsGenerated)
	}
}
