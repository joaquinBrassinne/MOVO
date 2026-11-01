package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// TimeToRedeemResult says how long redemptions took in a period. Each bucket is a Rate: the
// share of the redemptions of the period, against the same share in the previous period. With
// status insufficient_data only CouponsGenerated and Minimum are meaningful.
type TimeToRedeemResult struct {
	Status           string
	Period           benefits.MetricsPeriod
	Previous         benefits.MetricsPeriod
	CouponsGenerated int64
	Minimum          int

	Redemptions         int64
	PreviousRedemptions int64
	SameDay             Rate
	Within24h           Rate
	Within48h           Rate
}

// GetTimeToRedeem answers how fast coupons get redeemed (SDD 6.11), a Pro metric: the same day,
// within 24 hours or within 48 hours of being generated.
type GetTimeToRedeem struct {
	gate   ProGate
	reader benefits.TimeToRedeemReader
	now    func() time.Time
}

// NewGetTimeToRedeem wires the query. A nil clock uses time.Now.
func NewGetTimeToRedeem(gate ProGate, reader benefits.TimeToRedeemReader, now func() time.Time) *GetTimeToRedeem {
	if now == nil {
		now = time.Now
	}
	return &GetTimeToRedeem{gate: gate, reader: reader, now: now}
}

// Handle checks the plan first, then reads the period asked for (7, 30 or 90 days, 30 by default).
func (q *GetTimeToRedeem) Handle(ctx context.Context, ownerID int64, days int) (TimeToRedeemResult, error) {
	businessID, err := q.gate.Business(ctx, ownerID)
	if err != nil {
		return TimeToRedeemResult{}, err
	}
	days, err = validDays(days)
	if err != nil {
		return TimeToRedeemResult{}, err
	}

	current := periodOf(q.now(), days)
	previous := previousPeriod(current)
	got, err := q.reader.TimeToRedeem(ctx, businessID, current, previous)
	if err != nil {
		return TimeToRedeemResult{}, fmt.Errorf("read time to redeem: %w", err)
	}

	result := TimeToRedeemResult{
		Status: StatusOK, Period: current, Previous: previous,
		CouponsGenerated: got.CouponsGenerated, Minimum: benefits.MinimumSample,
	}
	if got.CouponsGenerated < benefits.MinimumSample {
		result.Status = StatusInsufficientData
		return result, nil
	}

	comparable := got.PreviousCouponsGenerated >= benefits.MinimumSample
	total, before := got.Current.Total(), got.Previous.Total()
	result.Redemptions, result.PreviousRedemptions = total, before
	result.SameDay = rate(got.Current.SameDay, total, got.Previous.SameDay, before, comparable)
	result.Within24h = rate(got.Current.Within24, total, got.Previous.Within24, before, comparable)
	result.Within48h = rate(got.Current.Within48, total, got.Previous.Within48, before, comparable)
	return result, nil
}
