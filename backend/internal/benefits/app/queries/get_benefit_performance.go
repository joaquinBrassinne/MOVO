package queries

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// BenefitPerformanceRow is how one benefit did: the coupons generated in the period and what
// became of them. Rate is the redeemed share of the generated (nil with none generated);
// Previous and Change follow the rules of Rate in the conversion metric.
type BenefitPerformanceRow struct {
	BenefitID int64
	Name      string
	Generated int64
	Redeemed  int64
	Expired   int64
	Pending   int64
	Rate      *float64
	Previous  *float64
	Change    *float64
}

// BenefitPerformanceResult is the performance of every benefit that generated a coupon in the
// period. With status insufficient_data only CouponsGenerated and Minimum are meaningful.
type BenefitPerformanceResult struct {
	Status           string
	Period           benefits.MetricsPeriod
	Previous         benefits.MetricsPeriod
	CouponsGenerated int64
	Minimum          int
	Benefits         []BenefitPerformanceRow
}

// GetBenefitPerformance answers how each benefit did (SDD 6.10), a Pro metric: coupons
// generated, redeemed and expired, and the redemption rate against the previous period.
type GetBenefitPerformance struct {
	gate     ProGate
	benefits benefits.BenefitRepository
	reader   benefits.PerformanceReader
	now      func() time.Time
}

// NewGetBenefitPerformance wires the query. A nil clock uses time.Now.
func NewGetBenefitPerformance(gate ProGate, repo benefits.BenefitRepository, reader benefits.PerformanceReader, now func() time.Time) *GetBenefitPerformance {
	if now == nil {
		now = time.Now
	}
	return &GetBenefitPerformance{gate: gate, benefits: repo, reader: reader, now: now}
}

// Handle checks the plan first, then reads the period asked for (7, 30 or 90 days, 30 by default).
func (q *GetBenefitPerformance) Handle(ctx context.Context, ownerID int64, days int) (BenefitPerformanceResult, error) {
	businessID, err := q.gate.Business(ctx, ownerID)
	if err != nil {
		return BenefitPerformanceResult{}, err
	}
	days, err = validDays(days)
	if err != nil {
		return BenefitPerformanceResult{}, err
	}

	now := q.now()
	current := periodOf(now, days)
	previous := previousPeriod(current)
	got, err := q.reader.BenefitPerformance(ctx, businessID, current, previous, now)
	if err != nil {
		return BenefitPerformanceResult{}, fmt.Errorf("read performance: %w", err)
	}

	var generated, previousGenerated int64
	for _, c := range got.Current {
		generated += c.Generated
	}
	for _, c := range got.Previous {
		previousGenerated += c.Generated
	}
	result := BenefitPerformanceResult{
		Status: StatusOK, Period: current, Previous: previous, CouponsGenerated: generated, Minimum: benefits.MinimumSample,
	}
	if generated < benefits.MinimumSample {
		result.Status = StatusInsufficientData
		return result, nil
	}

	owned, err := q.benefits.List(ctx, businessID)
	if err != nil {
		return BenefitPerformanceResult{}, fmt.Errorf("list benefits: %w", err)
	}
	names := make(map[int64]string, len(owned))
	for _, b := range owned {
		names[b.ID] = b.Title
		if b.Title == "" {
			names[b.ID] = b.ShortLabel()
		}
	}
	before := make(map[int64]benefits.BenefitCounts, len(got.Previous))
	for _, c := range got.Previous {
		before[c.BenefitID] = c
	}

	comparable := previousGenerated >= benefits.MinimumSample
	result.Benefits = make([]BenefitPerformanceRow, 0, len(got.Current))
	for _, c := range got.Current {
		row := BenefitPerformanceRow{
			BenefitID: c.BenefitID, Name: names[c.BenefitID], Generated: c.Generated, Redeemed: c.Redeemed,
			Expired: c.Expired, Pending: c.Generated - c.Redeemed - c.Expired, Rate: ratio(c.Redeemed, c.Generated),
		}
		if prev, ok := before[c.BenefitID]; ok && comparable {
			row.Previous = ratio(prev.Redeemed, prev.Generated)
		}
		if row.Rate != nil && row.Previous != nil {
			change := round4(*row.Rate - *row.Previous)
			row.Change = &change
		}
		result.Benefits = append(result.Benefits, row)
	}
	sort.SliceStable(result.Benefits, func(i, j int) bool {
		a, b := result.Benefits[i], result.Benefits[j]
		if a.Generated != b.Generated {
			return a.Generated > b.Generated
		}
		return a.BenefitID < b.BenefitID
	})
	return result, nil
}
