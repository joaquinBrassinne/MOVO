package benefitshttp

import (
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// quotaProjectionJSON carries what each status needs: no_quota is only the status; the rest
// add where the month stands; an estimate adds the pace and, when the quota runs out, the day.
type quotaProjectionJSON struct {
	Status      string `json:"status"`
	Limit       *int   `json:"limit,omitempty"`
	Generated   *int   `json:"generated,omitempty"`
	DaysElapsed *int   `json:"days_elapsed,omitempty"`
	DaysInMonth *int   `json:"days_in_month,omitempty"`
	// MinimumDays is how many days must pass before an estimate is made (insufficient_data).
	MinimumDays    *int     `json:"minimum_days,omitempty"`
	DailyRate      *float64 `json:"daily_rate,omitempty"`
	ProjectedTotal *int     `json:"projected_total,omitempty"`
	ExhaustsOn     *string  `json:"exhausts_on,omitempty"`
	ResetsOn       string   `json:"resets_on,omitempty"`
}

// handleQuotaProjection answers GET /me/business/metrics/quota-projection (Pro).
func (h *Handlers) handleQuotaProjection(w http.ResponseWriter, r *http.Request) {
	got, err := h.QuotaProjection.Handle(r.Context(), callerID(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}

	out := quotaProjectionJSON{Status: got.Status}
	if got.Status != domain.ProjectionNoQuota {
		out.Limit, out.Generated = &got.Limit, &got.Generated
		out.DaysElapsed, out.DaysInMonth = &got.DaysElapsed, &got.DaysInMonth
		out.ResetsOn = got.ResetsOn
	}
	switch got.Status {
	case domain.ProjectionTooEarly:
		minimum := domain.MinProjectionDays
		out.MinimumDays = &minimum
	case domain.ProjectionOnTrack, domain.ProjectionWillRunOut:
		out.DailyRate, out.ProjectedTotal = &got.DailyRate, &got.ProjectedTotal
	}
	if got.ExhaustsOn != "" {
		out.ExhaustsOn = &got.ExhaustsOn
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
