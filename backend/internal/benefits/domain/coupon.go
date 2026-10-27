package domain

import "time"

// CouponTTL is how long a coupon can be redeemed after it is generated (SDD 5.2).
const CouponTTL = 48 * time.Hour

// Status is where a coupon is in its life.
type Status string

// Coupon statuses (SDD 4).
const (
	StatusPending  Status = "pending"
	StatusRedeemed Status = "redeemed"
	StatusExpired  Status = "expired"
)

// Coupon is a code a customer generated to claim a benefit at the business.
type Coupon struct {
	ID         int64
	BenefitID  int64
	BusinessID int64
	CustomerID int64
	Code       string
	Status     Status
	ExpiresAt  time.Time
}

// ExpiryFrom returns when a coupon generated at now stops being redeemable.
func ExpiryFrom(now time.Time) time.Time {
	return now.Add(CouponTTL)
}

// StatusAt returns the status a reader must show at now. Expiry is evaluated on read:
// the stored status of an overdue coupon stays pending until the daily sweep marks it,
// but it is not redeemable from ExpiresAt on. A redeemed coupon stays redeemed.
func (c Coupon) StatusAt(now time.Time) Status {
	if c.Status == StatusPending && !now.Before(c.ExpiresAt) {
		return StatusExpired
	}
	return c.Status
}

// Redeemable reports whether the coupon can be redeemed at now. It mirrors the atomic
// UPDATE that redeems (status pending and expires_at in the future).
func (c Coupon) Redeemable(now time.Time) bool {
	return c.StatusAt(now) == StatusPending
}
