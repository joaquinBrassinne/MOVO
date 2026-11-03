package benefitshttp

import (
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// repeatCustomersJSON adds the counts once the sample is big enough. They are numbers: the
// answer never says who the customers are.
type repeatCustomersJSON struct {
	sampleJSON
	Customers         int64     `json:"customers"`
	PreviousCustomers int64     `json:"previous_customers"`
	OneRedemption     int64     `json:"one_redemption"`
	TwoOrMore         int64     `json:"two_or_more"`
	RepeatRate        *rateJSON `json:"repeat_rate"`
}

// handleRepeatCustomers answers GET /me/business/metrics/returning-customers?period=7|30|90 (Pro).
func (h *Handlers) handleRepeatCustomers(w http.ResponseWriter, r *http.Request) {
	days, ok := h.periodParam(w, r)
	if !ok {
		return
	}
	got, err := h.Returning.Handle(r.Context(), callerID(r), days)
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
	httpx.WriteJSON(w, http.StatusOK, repeatCustomersJSON{
		sampleJSON: sample, Customers: got.Customers, PreviousCustomers: got.PreviousCustomers,
		OneRedemption: got.OneRedemption, TwoOrMore: got.TwoOrMore, RepeatRate: toRateJSON(got.Repeat),
	})
}
