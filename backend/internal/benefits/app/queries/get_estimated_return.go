package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// StatusNoTicket means the owner never loaded an average ticket, so the estimate is not shown.
const StatusNoTicket = "no_ticket"

// EstimatedReturnResult is what the coupons of a period are estimated to have brought in. It is
// always an estimate: the redemptions of the period times the average ticket the owner loaded,
// as if every redemption had spent that much. Platform data cannot tell what a customer really
// spent. With status no_ticket or insufficient_data nothing but the period is meaningful.
type EstimatedReturnResult struct {
	Status           string
	Period           benefits.MetricsPeriod
	Previous         benefits.MetricsPeriod
	CouponsGenerated int64
	Minimum          int

	Redemptions         int64
	PreviousRedemptions int64
	AverageTicketARS    int
	EstimatedARS        int64
	// PreviousEstimatedARS is the same calculation for the previous period with the same ticket,
	// nil when that period had too few coupons to compare with.
	PreviousEstimatedARS *int64
	// PlanPriceARS is the monthly price of the plan the business is on, to set the estimate beside.
	PlanPriceARS int
}

// GetEstimatedReturn answers the estimated return (SDD 6.14), a Pro metric.
type GetEstimatedReturn struct {
	gate      ProGate
	economics benefits.BusinessEconomics
	reader    benefits.RedemptionTotalsReader
	now       func() time.Time
}

// NewGetEstimatedReturn wires the query. A nil clock uses time.Now.
func NewGetEstimatedReturn(gate ProGate, economics benefits.BusinessEconomics, reader benefits.RedemptionTotalsReader, now func() time.Time) *GetEstimatedReturn {
	if now == nil {
		now = time.Now
	}
	return &GetEstimatedReturn{gate: gate, economics: economics, reader: reader, now: now}
}

// Handle checks the plan first, then the ticket: without one nothing is read or shown.
func (q *GetEstimatedReturn) Handle(ctx context.Context, ownerID int64, days int) (EstimatedReturnResult, error) {
	businessID, err := q.gate.Business(ctx, ownerID)
	if err != nil {
		return EstimatedReturnResult{}, err
	}
	days, err = validDays(days)
	if err != nil {
		return EstimatedReturnResult{}, err
	}

	current := periodOf(q.now(), days)
	previous := previousPeriod(current)
	result := EstimatedReturnResult{Status: StatusOK, Period: current, Previous: previous, Minimum: benefits.MinimumSample}

	econ, err := q.economics.Economics(ctx, businessID)
	if err != nil {
		return EstimatedReturnResult{}, fmt.Errorf("read economics: %w", err)
	}
	if econ.AverageTicketARS == nil {
		result.Status = StatusNoTicket
		return result, nil
	}

	totals, err := q.reader.RedemptionTotals(ctx, businessID, current, previous)
	if err != nil {
		return EstimatedReturnResult{}, fmt.Errorf("read redemptions: %w", err)
	}
	result.CouponsGenerated = totals.CouponsGenerated
	if totals.CouponsGenerated < benefits.MinimumSample {
		result.Status = StatusInsufficientData
		return result, nil
	}

	ticket := int64(*econ.AverageTicketARS)
	result.Redemptions, result.PreviousRedemptions = totals.Redemptions, totals.PreviousRedemptions
	result.AverageTicketARS = *econ.AverageTicketARS
	result.EstimatedARS = totals.Redemptions * ticket
	result.PlanPriceARS = econ.PlanPriceARS
	if totals.PreviousCouponsGenerated >= benefits.MinimumSample {
		before := totals.PreviousRedemptions * ticket
		result.PreviousEstimatedARS = &before
	}
	return result, nil
}
