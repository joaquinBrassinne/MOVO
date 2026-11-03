package commands

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// alertWindowDays is how far back the redemption rate looks, today included.
const alertWindowDays = 30

// alertEvery is the least time between two alerts of this type for the same business.
const alertEvery = 7 * 24 * time.Hour

// RaiseUnredeemedAlerts is the daily task of the "coupons not redeemed" alert (SDD 4.3). It
// tells the owners of businesses with the Pro metrics whose redemption rate of the last 30
// days is under the threshold, once there are enough coupons for the rate to mean something,
// and at most once a week each.
type RaiseUnredeemedAlerts struct {
	scanner       benefits.LowRedemptionScanner
	permission    benefits.MetricsPermission
	alerts        benefits.AlertRaiser
	minCoupons    int
	minRedeemRate float64
	now           func() time.Time
}

// NewRaiseUnredeemedAlerts wires the use case. minCoupons and minRedeemRate come from the
// environment. A nil clock uses time.Now.
func NewRaiseUnredeemedAlerts(scanner benefits.LowRedemptionScanner, permission benefits.MetricsPermission, alerts benefits.AlertRaiser, minCoupons int, minRedeemRate float64, now func() time.Time) *RaiseUnredeemedAlerts {
	if now == nil {
		now = time.Now
	}
	return &RaiseUnredeemedAlerts{scanner: scanner, permission: permission, alerts: alerts, minCoupons: minCoupons, minRedeemRate: minRedeemRate, now: now}
}

// Handle raises the alerts due today and returns how many it stored. It is safe to repeat.
func (c *RaiseUnredeemedAlerts) Handle(ctx context.Context) (int, error) {
	now := c.now()
	todayStart, todayEnd := domain.DayInArgentina(now)
	from := todayStart.AddDate(0, 0, -(alertWindowDays - 1))

	rates, err := c.scanner.LowRedemption(ctx, from, todayEnd, c.minCoupons, c.minRedeemRate)
	if err != nil {
		return 0, fmt.Errorf("raise unredeemed alerts: %w", err)
	}
	raised := 0
	for _, r := range rates {
		pro, err := c.permission.AdvancedMetrics(ctx, r.BusinessID)
		if err != nil {
			return raised, fmt.Errorf("raise unredeemed alerts: plan of business %d: %w", r.BusinessID, err)
		}
		if !pro {
			continue
		}
		payload := map[string]any{
			"period_days":       alertWindowDays,
			"coupons_generated": r.Generated,
			"coupons_redeemed":  r.Redeemed,
			"redeem_rate":       math.Round(float64(r.Redeemed)/float64(r.Generated)*10000) / 10000,
			"threshold":         c.minRedeemRate,
		}
		stored, err := c.alerts.Raise(ctx, r.BusinessID, benefits.AlertUnredeemedCoupons, payload, now.Add(-alertEvery))
		if err != nil {
			return raised, fmt.Errorf("raise unredeemed alerts: business %d: %w", r.BusinessID, err)
		}
		if stored {
			raised++
		}
	}
	return raised, nil
}
