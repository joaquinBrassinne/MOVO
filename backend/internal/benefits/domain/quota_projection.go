package domain

import (
	"math"
	"time"
)

// MinProjectionDays is how many days of the month must have gone by before a projection says
// anything: with fewer, a single busy day would predict the end of the month.
const MinProjectionDays = 3

// Where the monthly quota stands.
const (
	// ProjectionNoQuota: the business has no limit, so there is nothing to run out of.
	ProjectionNoQuota = "no_quota"
	// ProjectionTooEarly: not enough of the month has gone by, or nothing was generated yet.
	ProjectionTooEarly = "insufficient_data"
	// ProjectionReached: the quota is already used up.
	ProjectionReached = "exhausted"
	// ProjectionOnTrack: at the pace of the month so far, the quota lasts until it resets.
	ProjectionOnTrack = "on_track"
	// ProjectionWillRunOut: at the pace of the month so far, the quota runs out before it resets.
	ProjectionWillRunOut = "will_run_out"
)

// QuotaProjection is a linear estimate of the day the monthly quota runs out, from the pace of
// the month so far (SDD 6.13). It is only an estimate: it assumes the next days look like the
// ones that went by.
type QuotaProjection struct {
	Status      string
	Limit       int
	Generated   int
	DaysElapsed int
	DaysInMonth int
	// DailyRate is the coupons generated per day so far, and ProjectedTotal what the month
	// would end at keeping that pace. Both are set when the estimate could be made.
	DailyRate      float64
	ProjectedTotal int
	// ExhaustsOn is the Argentine calendar day the quota would run out (YYYY-MM-DD), set only
	// for ProjectionWillRunOut.
	ExhaustsOn string
}

// ProjectQuota estimates where the quota of the month [monthStart, monthEnd) stands at now.
// Today counts as a day gone by, so the pace is the coupons generated over the day of the month.
func ProjectQuota(q Quota, monthStart, monthEnd, now time.Time) QuotaProjection {
	elapsed := now.In(argentina).Day()
	daysInMonth := int(monthEnd.Sub(monthStart).Hours() / 24)
	out := QuotaProjection{Generated: q.Generated, DaysElapsed: elapsed, DaysInMonth: daysInMonth}

	if q.Limit == nil {
		out.Status = ProjectionNoQuota
		return out
	}
	out.Limit = *q.Limit
	switch {
	case q.Generated >= *q.Limit:
		out.Status = ProjectionReached
		return out
	case elapsed < MinProjectionDays || q.Generated == 0:
		out.Status = ProjectionTooEarly
		return out
	}

	rate := float64(q.Generated) / float64(elapsed)
	out.DailyRate = math.Round(rate*100) / 100
	out.ProjectedTotal = int(math.Round(rate * float64(daysInMonth)))

	remaining := float64(*q.Limit - q.Generated)
	lastDay := elapsed + int(math.Ceil(remaining/rate))
	if lastDay > daysInMonth {
		out.Status = ProjectionOnTrack
		return out
	}
	out.Status = ProjectionWillRunOut
	out.ExhaustsOn = monthStart.AddDate(0, 0, lastDay-1).Format(time.DateOnly)
	return out
}
