package postgres_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/postgres"
)

func TestRaiseStoresAnAlertAtMostOncePerTypeInTheWindow(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	business, other := f.business(t, "active"), f.business(t, "active")
	alerts := postgres.NewAlerts(f.pool)
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	payload := map[string]any{"coupons_generated": 40, "redeem_rate": 0.1}

	stored, err := alerts.Raise(t.Context(), business, "unredeemed_coupons", payload, weekAgo)
	if err != nil || !stored {
		t.Fatalf("first Raise = %t, %v; want stored", stored, err)
	}
	if stored, err := alerts.Raise(t.Context(), business, "unredeemed_coupons", payload, weekAgo); err != nil || stored {
		t.Errorf("second Raise = %t, %v; want it skipped, one this week already", stored, err)
	}
	if stored, err := alerts.Raise(t.Context(), other, "unredeemed_coupons", payload, weekAgo); err != nil || !stored {
		t.Errorf("another business = %t, %v; want stored", stored, err)
	}
	if stored, err := alerts.Raise(t.Context(), business, "plan_downgrade", payload, weekAgo); err != nil || !stored {
		t.Errorf("another type = %t, %v; want stored", stored, err)
	}

	// Read or not, the first one still counts for the week; once it is older, a new one is due.
	if _, err := f.pool.Exec(t.Context(), `UPDATE alerts SET created_at = now() - interval '8 days', read_at = now() WHERE business_id = $1 AND type = 'unredeemed_coupons'`, business); err != nil {
		t.Fatal(err)
	}
	if stored, err := alerts.Raise(t.Context(), business, "unredeemed_coupons", payload, time.Now().Add(-7*24*time.Hour)); err != nil || !stored {
		t.Errorf("after 8 days = %t, %v; want a new alert", stored, err)
	}

	var rate float64
	if err := f.pool.QueryRow(t.Context(), `SELECT (payload->>'redeem_rate')::float8 FROM alerts WHERE business_id = $1 AND type = 'unredeemed_coupons' ORDER BY id DESC LIMIT 1`, business).Scan(&rate); err != nil || rate != 0.1 {
		t.Errorf("payload rate = %v, %v; want 0.1", rate, err)
	}
}
