// Package queries holds the super admin's read use cases.
package queries

import (
	"context"
	"fmt"
	"time"
	_ "time/tzdata" // the zone must resolve on hosts without a system tz database

	"github.com/joaquinBrassinne/MOVO/internal/admin"
)

// zoneName is the zone the pilot lives in; the day and the month roll over there.
const zoneName = "America/Argentina/Cordoba"

// zone has no daylight saving since 2009, so the fixed offset is only the fallback.
var zone = func() *time.Location {
	if loc, err := time.LoadLocation(zoneName); err == nil {
		return loc
	}
	return time.FixedZone("ART", -3*60*60)
}()

// GetOverview answers the figures of the super admin's dashboard.
type GetOverview struct {
	reader admin.OverviewReader
	now    func() time.Time
}

// NewGetOverview wires the query. A nil now uses time.Now.
func NewGetOverview(reader admin.OverviewReader, now func() time.Time) *GetOverview {
	if now == nil {
		now = time.Now
	}
	return &GetOverview{reader: reader, now: now}
}

// Handle reads the overview with the current month in Argentina.
func (q *GetOverview) Handle(ctx context.Context) (admin.Overview, error) {
	monthStart := MonthStart(q.now())
	overview, err := q.reader.Overview(ctx, monthStart)
	if err != nil {
		return admin.Overview{}, fmt.Errorf("read overview: %w", err)
	}
	overview.MonthStart = monthStart
	return overview, nil
}

// MonthStart is the first instant of the month that t falls in, on Argentina's clock.
func MonthStart(t time.Time) time.Time {
	local := t.In(zone)
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, zone)
}
