package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// DefaultMetricsDays is the period the panel opens with.
const DefaultMetricsDays = 30

// MetricsResult is the metrics of the owner's business over the period they asked for.
type MetricsResult struct {
	Period  benefits.MetricsPeriod
	Metrics benefits.EssentialMetrics
}

// GetMetrics answers the essential metrics of the owner's business (SDD 6.4).
type GetMetrics struct {
	owners  benefits.OwnerBusinesses
	metrics benefits.MetricsReader
	now     func() time.Time
}

// NewGetMetrics wires the query. A nil clock uses time.Now.
func NewGetMetrics(owners benefits.OwnerBusinesses, metrics benefits.MetricsReader, now func() time.Time) *GetMetrics {
	if now == nil {
		now = time.Now
	}
	return &GetMetrics{owners: owners, metrics: metrics, now: now}
}

// Handle reads the last days days, today included. days is 7, 30 or 90, and 0 means
// DefaultMetricsDays; any other value is benefits.ErrMetricsPeriodInvalid. The business
// comes from the session.
func (q *GetMetrics) Handle(ctx context.Context, ownerID int64, days int) (MetricsResult, error) {
	days, err := validDays(days)
	if err != nil {
		return MetricsResult{}, err
	}
	businessID, err := q.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return MetricsResult{}, fmt.Errorf("resolve business: %w", err)
	}

	period := periodOf(q.now(), days)
	got, err := q.metrics.Essential(ctx, businessID, period)
	if err != nil {
		return MetricsResult{}, fmt.Errorf("read metrics: %w", err)
	}
	return MetricsResult{Period: period, Metrics: got}, nil
}

// validDays applies the closed list of periods: 7, 30 or 90 days, and 0 for the default.
func validDays(days int) (int, error) {
	if days == 0 {
		return DefaultMetricsDays, nil
	}
	if days != 7 && days != 30 && days != 90 {
		return 0, benefits.ErrMetricsPeriodInvalid
	}
	return days, nil
}

// periodOf is the window of the last days days that ends with the day of now.
func periodOf(now time.Time, days int) benefits.MetricsPeriod {
	todayStart, todayEnd := domain.DayInArgentina(now)
	from := todayStart.AddDate(0, 0, -(days - 1))
	return benefits.MetricsPeriod{
		Days: days, From: from, To: todayEnd,
		FromDay: domain.DateInArgentina(from), ToDay: domain.DateInArgentina(todayStart),
	}
}
