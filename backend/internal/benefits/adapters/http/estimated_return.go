package benefitshttp

import (
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// estimatedReturnJSON adds the figures once there is a ticket and enough coupons. Estimated is
// always true: the screen must say so, and the formula is the redemptions times the ticket.
type estimatedReturnJSON struct {
	sampleJSON
	Estimated            bool   `json:"estimated"`
	Redemptions          int64  `json:"redemptions"`
	PreviousRedemptions  int64  `json:"previous_redemptions"`
	AverageTicketARS     int    `json:"average_ticket_ars"`
	EstimatedReturnARS   int64  `json:"estimated_return_ars"`
	PreviousEstimatedARS *int64 `json:"previous_estimated_return_ars"`
	PlanPriceARS         int    `json:"plan_price_ars"`
}

// handleEstimatedReturn answers GET /me/business/metrics/estimated-return?period=7|30|90 (Pro).
func (h *Handlers) handleEstimatedReturn(w http.ResponseWriter, r *http.Request) {
	days, ok := h.periodParam(w, r)
	if !ok {
		return
	}
	got, err := h.EstimatedReturn.Handle(r.Context(), callerID(r), days)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	sample := sampleJSON{
		Status: got.Status, Period: toPeriodJSON(got.Period), PreviousPeriod: toPeriodJSON(got.Previous),
		CouponsGenerated: got.CouponsGenerated, MinimumSample: got.Minimum,
	}
	if got.Status != queries.StatusOK {
		httpx.WriteJSON(w, http.StatusOK, sample)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, estimatedReturnJSON{
		sampleJSON: sample, Estimated: true,
		Redemptions: got.Redemptions, PreviousRedemptions: got.PreviousRedemptions,
		AverageTicketARS: got.AverageTicketARS, EstimatedReturnARS: got.EstimatedARS,
		PreviousEstimatedARS: got.PreviousEstimatedARS, PlanPriceARS: got.PlanPriceARS,
	})
}
