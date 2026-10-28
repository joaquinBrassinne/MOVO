package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// ActivityLimit is how many events the panel lists.
const ActivityLimit = 20

// RecentActivity feeds the owner's "Actividad reciente" (SDD 5.6). The panel asks again
// every 30-60 seconds, so it is a plain read.
type RecentActivity struct {
	owners   benefits.OwnerBusinesses
	activity benefits.ActivityReader
}

// NewRecentActivity wires the RecentActivity query.
func NewRecentActivity(owners benefits.OwnerBusinesses, activity benefits.ActivityReader) *RecentActivity {
	return &RecentActivity{owners: owners, activity: activity}
}

// Handle returns the newest events of the owner's business and how many are newer than
// since, the moment the panel last showed the news. The business comes from the session.
func (q *RecentActivity) Handle(ctx context.Context, ownerID int64, since *time.Time) (benefits.Activity, error) {
	businessID, err := q.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return benefits.Activity{}, fmt.Errorf("resolve business: %w", err)
	}
	got, err := q.activity.RecentActivity(ctx, businessID, since, ActivityLimit)
	if err != nil {
		return benefits.Activity{}, fmt.Errorf("read activity: %w", err)
	}
	return got, nil
}
