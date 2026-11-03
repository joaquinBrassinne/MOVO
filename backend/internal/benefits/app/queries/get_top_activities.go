package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// TopActivitiesLimit is how many activities the ranking shows.
const TopActivitiesLimit = 5

// TopActivitiesResult is the ranking of the owner's activities over a period.
type TopActivitiesResult struct {
	Period     benefits.MetricsPeriod
	Activities []benefits.ActivityDemand
}

// GetTopActivities answers the most demanded activities of the owner's business (SDD 6.5):
// the views of each service plus the taps on WhatsApp from it.
type GetTopActivities struct {
	owners benefits.OwnerBusinesses
	demand benefits.DemandReader
	now    func() time.Time
}

// NewGetTopActivities wires the query. A nil clock uses time.Now.
func NewGetTopActivities(owners benefits.OwnerBusinesses, demand benefits.DemandReader, now func() time.Time) *GetTopActivities {
	if now == nil {
		now = time.Now
	}
	return &GetTopActivities{owners: owners, demand: demand, now: now}
}

// Handle ranks the last days days, with the same closed list of periods as GetMetrics.
func (q *GetTopActivities) Handle(ctx context.Context, ownerID int64, days int) (TopActivitiesResult, error) {
	days, err := validDays(days)
	if err != nil {
		return TopActivitiesResult{}, err
	}
	businessID, err := q.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return TopActivitiesResult{}, fmt.Errorf("resolve business: %w", err)
	}

	period := periodOf(q.now(), days)
	got, err := q.demand.TopActivities(ctx, businessID, period, TopActivitiesLimit)
	if err != nil {
		return TopActivitiesResult{}, fmt.Errorf("read demand: %w", err)
	}
	if got == nil {
		got = []benefits.ActivityDemand{}
	}
	return TopActivitiesResult{Period: period, Activities: got}, nil
}
