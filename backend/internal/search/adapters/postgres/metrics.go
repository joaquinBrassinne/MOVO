package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// Metrics implements search.MetricCounter.
type Metrics struct {
	pool *pgxpool.Pool
}

// NewMetrics builds the counter over pool.
func NewMetrics(pool *pgxpool.Pool) *Metrics {
	return &Metrics{pool: pool}
}

// incrementSQL adds one in a single statement, so two requests at once both count.
// Zero branch and service become NULL; the unique key treats NULLs as equal.
const incrementSQL = `
INSERT INTO metric_daily (business_id, branch_id, service_id, date, metric, count)
VALUES ($1, NULLIF($2::bigint, 0), NULLIF($3::bigint, 0), $4::date, $5, 1)
ON CONFLICT ON CONSTRAINT metric_daily_key DO UPDATE SET count = metric_daily.count + 1`

// Increment counts e on the calendar day of day. It validates the event, so a shape
// the database would reject never reaches it.
func (m *Metrics) Increment(ctx context.Context, e search.Event, day time.Time) error {
	if err := e.Validate(); err != nil {
		return err
	}
	if _, err := m.pool.Exec(ctx, incrementSQL,
		e.BusinessID, e.BranchID, e.ServiceID, day.Format(time.DateOnly), string(e.Metric)); err != nil {
		return fmt.Errorf("postgres: increment %s: %w", e.Metric, err)
	}
	return nil
}
