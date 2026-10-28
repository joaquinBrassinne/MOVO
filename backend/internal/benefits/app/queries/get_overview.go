package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// GetOverview feeds the home of the owner's panel with a few counts. Anything beyond them
// (trends, breakdowns) is the metrics of phase 6.
type GetOverview struct {
	owners   benefits.OwnerBusinesses
	overview benefits.OverviewReader
	now      func() time.Time
}

// NewGetOverview wires the GetOverview query. A nil clock uses time.Now.
func NewGetOverview(owners benefits.OwnerBusinesses, overview benefits.OverviewReader, now func() time.Time) *GetOverview {
	if now == nil {
		now = time.Now
	}
	return &GetOverview{owners: owners, overview: overview, now: now}
}

// Handle returns the counts of the owner's business. The business comes from the session.
func (q *GetOverview) Handle(ctx context.Context, ownerID int64) (benefits.Overview, error) {
	businessID, err := q.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return benefits.Overview{}, fmt.Errorf("resolve business: %w", err)
	}
	got, err := q.overview.Overview(ctx, businessID, q.now())
	if err != nil {
		return benefits.Overview{}, fmt.Errorf("read overview: %w", err)
	}
	return got, nil
}
