package queries

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// Statuses of a Pro metric.
const (
	StatusOK               = "ok"
	StatusInsufficientData = "insufficient_data"
)

// Rate is a ratio with the counts it comes from and how it moved against the previous period.
// Value is nil when there is nothing to divide by. Previous and Change are nil when the
// previous period has too little data to compare with (fewer than benefits.MinimumSample
// coupons generated) or nothing to divide by. Change is Value minus Previous, as a fraction.
type Rate struct {
	Numerator   int64
	Denominator int64
	Value       *float64
	Previous    *float64
	Change      *float64
}

// BranchRate is the conversion of one branch: the coupons redeemed there per tap on its WhatsApp.
type BranchRate struct {
	BranchID       int64
	Name           string
	WhatsAppClicks int64
	Redemptions    int64
	Rate           *float64
}

// ActivityRate is the conversion of one activity: the taps on WhatsApp per view.
type ActivityRate struct {
	ServiceID      int64
	Name           string
	Views          int64
	WhatsAppClicks int64
	Rate           *float64
}

// ConversionResult is the conversion metric of a period. With status insufficient_data only
// CouponsGenerated and Minimum are meaningful: no rate is calculated.
type ConversionResult struct {
	Status           string
	Period           benefits.MetricsPeriod
	Previous         benefits.MetricsPeriod
	CouponsGenerated int64
	Minimum          int

	ViewsToCoupons       Rate
	CouponsToRedemptions Rate
	ByBranch             []BranchRate
	ByActivity           []ActivityRate
}

// GetConversion answers the conversion rates with their trend and breakdown (SDD 6.8), a Pro
// metric: views to coupons at the level of the business, coupons to redemptions, the
// redemptions per tap by branch and the taps per view by activity.
type GetConversion struct {
	gate   ProGate
	reader benefits.ConversionReader
	now    func() time.Time
}

// NewGetConversion wires the query. A nil clock uses time.Now.
func NewGetConversion(gate ProGate, reader benefits.ConversionReader, now func() time.Time) *GetConversion {
	if now == nil {
		now = time.Now
	}
	return &GetConversion{gate: gate, reader: reader, now: now}
}

// Handle checks the plan first, then reads the period asked for (7, 30 or 90 days, 30 by
// default) and the one before it.
func (q *GetConversion) Handle(ctx context.Context, ownerID int64, days int) (ConversionResult, error) {
	businessID, err := q.gate.Business(ctx, ownerID)
	if err != nil {
		return ConversionResult{}, err
	}
	days, err = validDays(days)
	if err != nil {
		return ConversionResult{}, err
	}

	current := periodOf(q.now(), days)
	previous := previousPeriod(current)
	got, err := q.reader.Conversion(ctx, businessID, current, previous)
	if err != nil {
		return ConversionResult{}, fmt.Errorf("read conversion: %w", err)
	}

	result := ConversionResult{
		Status: StatusOK, Period: current, Previous: previous,
		CouponsGenerated: got.Current.CouponsGenerated, Minimum: benefits.MinimumSample,
	}
	if got.Current.CouponsGenerated < benefits.MinimumSample {
		result.Status = StatusInsufficientData
		return result, nil
	}

	// A previous period with too little data is not a baseline.
	comparable := got.Previous.CouponsGenerated >= benefits.MinimumSample
	result.ViewsToCoupons = rate(got.Current.CouponsGenerated, got.Current.ProfileViews,
		got.Previous.CouponsGenerated, got.Previous.ProfileViews, comparable)
	result.CouponsToRedemptions = rate(got.Current.CouponsRedeemed, got.Current.CouponsGenerated,
		got.Previous.CouponsRedeemed, got.Previous.CouponsGenerated, comparable)

	result.ByBranch = make([]BranchRate, 0, len(got.ByBranch))
	for _, b := range got.ByBranch {
		result.ByBranch = append(result.ByBranch, BranchRate{
			BranchID: b.BranchID, Name: b.Name, WhatsAppClicks: b.WhatsAppClicks, Redemptions: b.Redemptions,
			Rate: ratio(b.Redemptions, b.WhatsAppClicks),
		})
	}
	result.ByActivity = make([]ActivityRate, 0, len(got.ByActivity))
	for _, a := range got.ByActivity {
		result.ByActivity = append(result.ByActivity, ActivityRate{
			ServiceID: a.ServiceID, Name: a.Name, Views: a.Views, WhatsAppClicks: a.WhatsAppClicks,
			Rate: ratio(a.WhatsAppClicks, a.Views),
		})
	}
	return result, nil
}

// previousPeriod is the window of the same length that ends where p begins.
func previousPeriod(p benefits.MetricsPeriod) benefits.MetricsPeriod {
	from := p.From.AddDate(0, 0, -p.Days)
	return benefits.MetricsPeriod{
		Days: p.Days, From: from, To: p.From,
		FromDay: from.In(p.From.Location()).Format(time.DateOnly),
		ToDay:   p.From.AddDate(0, 0, -1).Format(time.DateOnly),
	}
}

// rate builds a Rate from the counts of the period and of the one before it.
func rate(num, den, prevNum, prevDen int64, comparable bool) Rate {
	r := Rate{Numerator: num, Denominator: den, Value: ratio(num, den)}
	if comparable {
		r.Previous = ratio(prevNum, prevDen)
	}
	if r.Value != nil && r.Previous != nil {
		change := round4(*r.Value - *r.Previous)
		r.Change = &change
	}
	return r
}

// ratio is num/den as a fraction rounded to four decimals, nil when den is zero.
func ratio(num, den int64) *float64 {
	if den == 0 {
		return nil
	}
	v := round4(float64(num) / float64(den))
	return &v
}

func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
