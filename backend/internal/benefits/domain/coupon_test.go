package domain_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

func TestExpiryFromIs48Hours(t *testing.T) {
	t.Parallel()

	now := time.Date(2028, 3, 1, 10, 0, 0, 0, time.UTC)
	if got, want := domain.ExpiryFrom(now), now.Add(48*time.Hour); !got.Equal(want) {
		t.Errorf("ExpiryFrom = %v, want %v", got, want)
	}
}

func TestCouponStatusAndRedeemableAreEvaluatedOnRead(t *testing.T) {
	t.Parallel()

	expires := time.Date(2028, 3, 3, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		status     domain.Status
		now        time.Time
		wantStatus domain.Status
		wantOK     bool
	}{
		{"pending before the expiry", domain.StatusPending, expires.Add(-time.Second), domain.StatusPending, true},
		{"pending at the expiry", domain.StatusPending, expires, domain.StatusExpired, false},
		{"pending after the expiry, not yet swept", domain.StatusPending, expires.Add(time.Hour), domain.StatusExpired, false},
		{"already marked expired", domain.StatusExpired, expires.Add(-time.Hour), domain.StatusExpired, false},
		{"redeemed before the expiry", domain.StatusRedeemed, expires.Add(-time.Hour), domain.StatusRedeemed, false},
		{"redeemed stays redeemed after the expiry", domain.StatusRedeemed, expires.Add(time.Hour), domain.StatusRedeemed, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := domain.Coupon{Status: tt.status, ExpiresAt: expires}
			if got := c.StatusAt(tt.now); got != tt.wantStatus {
				t.Errorf("StatusAt = %q, want %q", got, tt.wantStatus)
			}
			if got := c.Redeemable(tt.now); got != tt.wantOK {
				t.Errorf("Redeemable = %v, want %v", got, tt.wantOK)
			}
		})
	}
}
