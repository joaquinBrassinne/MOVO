// Package admin is the super admin's read side: the platform-wide figures the panel shows.
// It only reads, so its port is a report reader and not a repository.
package admin

import (
	"context"
	"time"
)

// CouponCounts is how many coupons were generated and how many of them were redeemed.
type CouponCounts struct {
	Generated int
	Redeemed  int
}

// Overview is the platform at a glance (SDD 6.3). Counts carry no personal data.
type Overview struct {
	ActiveBusinesses int
	// Coupons is the all-time total.
	Coupons CouponCounts
	// CouponsThisMonth counts coupons generated, and coupons redeemed, since MonthStart.
	CouponsThisMonth CouponCounts
	MonthStart       time.Time
}

// OverviewReader reads the figures. monthStart is the first instant of the current month in
// Argentina, so "this month" means the month the pilot lives in, not the server's.
type OverviewReader interface {
	Overview(ctx context.Context, monthStart time.Time) (Overview, error)
}
