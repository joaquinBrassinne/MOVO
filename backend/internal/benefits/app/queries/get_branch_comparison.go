package queries

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// Count is a number with the same number in the previous period and how it moved. Change is
// the growth as a fraction (0.25 is +25 %), nil when there is no baseline: the previous period
// had too few coupons or the number was zero.
type Count struct {
	Value    int64
	Previous int64
	Change   *float64
}

// BranchRow is one branch compared: what its services, its WhatsApp button and its redemptions did.
type BranchRow struct {
	BranchID       int64
	Name           string
	Active         bool
	ServiceViews   Count
	WhatsAppClicks Count
	Redemptions    Count
}

// BranchComparisonResult compares the branches of a business over a period. With status
// insufficient_data only CouponsGenerated and Minimum are meaningful.
type BranchComparisonResult struct {
	Status           string
	Period           benefits.MetricsPeriod
	Previous         benefits.MetricsPeriod
	CouponsGenerated int64
	Minimum          int
	Branches         []BranchRow
}

// GetBranchComparison answers how each branch did (SDD 6.15), a Pro metric: views of services,
// taps on WhatsApp and redemptions per branch, each against the previous period.
type GetBranchComparison struct {
	gate   ProGate
	reader benefits.BranchComparisonReader
	now    func() time.Time
}

// NewGetBranchComparison wires the query. A nil clock uses time.Now.
func NewGetBranchComparison(gate ProGate, reader benefits.BranchComparisonReader, now func() time.Time) *GetBranchComparison {
	if now == nil {
		now = time.Now
	}
	return &GetBranchComparison{gate: gate, reader: reader, now: now}
}

// Handle checks the plan first, then reads the period asked for (7, 30 or 90 days, 30 by default).
func (q *GetBranchComparison) Handle(ctx context.Context, ownerID int64, days int) (BranchComparisonResult, error) {
	businessID, err := q.gate.Business(ctx, ownerID)
	if err != nil {
		return BranchComparisonResult{}, err
	}
	days, err = validDays(days)
	if err != nil {
		return BranchComparisonResult{}, err
	}

	current := periodOf(q.now(), days)
	previous := previousPeriod(current)
	got, err := q.reader.BranchComparison(ctx, businessID, current, previous)
	if err != nil {
		return BranchComparisonResult{}, fmt.Errorf("read branch comparison: %w", err)
	}

	result := BranchComparisonResult{
		Status: StatusOK, Period: current, Previous: previous,
		CouponsGenerated: got.CouponsGenerated, Minimum: benefits.MinimumSample,
	}
	if got.CouponsGenerated < benefits.MinimumSample {
		result.Status = StatusInsufficientData
		return result, nil
	}

	comparable := got.PreviousCouponsGenerated >= benefits.MinimumSample
	result.Branches = make([]BranchRow, 0, len(got.Branches))
	for _, b := range got.Branches {
		result.Branches = append(result.Branches, BranchRow{
			BranchID: b.BranchID, Name: b.Name, Active: b.Active,
			ServiceViews:   count(b.Current.ServiceViews, b.Previous.ServiceViews, comparable),
			WhatsAppClicks: count(b.Current.WhatsAppClicks, b.Previous.WhatsAppClicks, comparable),
			Redemptions:    count(b.Current.Redemptions, b.Previous.Redemptions, comparable),
		})
	}
	sort.SliceStable(result.Branches, func(i, j int) bool {
		a, b := result.Branches[i], result.Branches[j]
		if a.Redemptions.Value != b.Redemptions.Value {
			return a.Redemptions.Value > b.Redemptions.Value
		}
		return a.Name < b.Name
	})
	return result, nil
}

// count builds a Count, with a growth only when the previous period is a baseline.
func count(value, previous int64, comparable bool) Count {
	c := Count{Value: value, Previous: previous}
	if comparable && previous > 0 {
		growth := round4(float64(value-previous) / float64(previous))
		c.Change = &growth
	}
	return c
}
