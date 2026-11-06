package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// RepeatCustomersResult says how many customers came back in a period. Repeat is the share of
// the customers who redeemed that redeemed two or more times, against the previous period. With
// status insufficient_data only CouponsGenerated and Minimum are meaningful.
type RepeatCustomersResult struct {
	Status           string
	Period           benefits.MetricsPeriod
	Previous         benefits.MetricsPeriod
	CouponsGenerated int64
	Minimum          int

	Customers         int64
	PreviousCustomers int64
	OneRedemption     int64
	TwoOrMore         int64
	Repeat            Rate
}

// GetRepeatCustomers answers how many customers came back (SDD 6.12), a Pro metric: customers
// with one redemption against customers with two or more at the business, as numbers only.
type GetRepeatCustomers struct {
	gate   ProGate
	reader benefits.RepeatCustomersReader
	now    func() time.Time
}

// NewGetRepeatCustomers wires the query. A nil clock uses time.Now.
func NewGetRepeatCustomers(gate ProGate, reader benefits.RepeatCustomersReader, now func() time.Time) *GetRepeatCustomers {
	if now == nil {
		now = time.Now
	}
	return &GetRepeatCustomers{gate: gate, reader: reader, now: now}
}

// Handle checks the plan first, then reads the period asked for (7, 30 or 90 days, 30 by default).
func (q *GetRepeatCustomers) Handle(ctx context.Context, ownerID int64, days int) (RepeatCustomersResult, error) {
	businessID, err := q.gate.Business(ctx, ownerID)
	if err != nil {
		return RepeatCustomersResult{}, err
	}
	days, err = validDays(days)
	if err != nil {
		return RepeatCustomersResult{}, err
	}

	current := periodOf(q.now(), days)
	previous := previousPeriod(current)
	got, err := q.reader.RepeatCustomers(ctx, businessID, current, previous)
	if err != nil {
		return RepeatCustomersResult{}, fmt.Errorf("read repeat customers: %w", err)
	}

	result := RepeatCustomersResult{
		Status: StatusOK, Period: current, Previous: previous,
		CouponsGenerated: got.CouponsGenerated, Minimum: benefits.MinimumSample,
	}
	if got.CouponsGenerated < benefits.MinimumSample {
		result.Status = StatusInsufficientData
		return result, nil
	}

	comparable := got.PreviousCouponsGenerated >= benefits.MinimumSample
	result.Customers, result.PreviousCustomers = got.Current.Total(), got.Previous.Total()
	result.OneRedemption, result.TwoOrMore = got.Current.OneRedemption, got.Current.Repeat
	result.Repeat = rate(got.Current.Repeat, got.Current.Total(), got.Previous.Repeat, got.Previous.Total(), comparable)
	return result, nil
}
