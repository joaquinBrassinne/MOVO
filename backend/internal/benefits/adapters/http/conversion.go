package benefitshttp

import (
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

func toPeriodJSON(p benefits.MetricsPeriod) periodJSON {
	return periodJSON{Days: p.Days, From: p.FromDay, To: p.ToDay}
}

// rateJSON is a ratio with its counts and its movement against the previous period. The rates
// are fractions (0.25 is 25 %); they are null when there is nothing to divide by or compare with.
type rateJSON struct {
	Numerator    int64    `json:"numerator"`
	Denominator  int64    `json:"denominator"`
	Rate         *float64 `json:"rate"`
	PreviousRate *float64 `json:"previous_rate"`
	Change       *float64 `json:"change"`
}

func toRateJSON(r queries.Rate) *rateJSON {
	return &rateJSON{Numerator: r.Numerator, Denominator: r.Denominator, Rate: r.Value, PreviousRate: r.Previous, Change: r.Change}
}

type branchRateJSON struct {
	BranchID       int64    `json:"branch_id"`
	Name           string   `json:"name"`
	WhatsAppClicks int64    `json:"whatsapp_clicks"`
	Redemptions    int64    `json:"redemptions"`
	Rate           *float64 `json:"rate"`
}

type activityRateJSON struct {
	ServiceID      int64    `json:"service_id"`
	Name           string   `json:"name"`
	Views          int64    `json:"views"`
	WhatsAppClicks int64    `json:"whatsapp_clicks"`
	Rate           *float64 `json:"rate"`
}

// sampleJSON is what every answer of the conversion metric carries, and all there is when the
// sample is too small: the period, how many coupons it has and how many it needs.
type sampleJSON struct {
	Status           string     `json:"status"`
	Period           periodJSON `json:"period"`
	PreviousPeriod   periodJSON `json:"previous_period"`
	CouponsGenerated int64      `json:"coupons_generated"`
	MinimumSample    int        `json:"minimum_sample"`
}

// conversionJSON adds the rates and the breakdown once the sample is big enough.
type conversionJSON struct {
	sampleJSON
	ViewsToCoupons       *rateJSON          `json:"views_to_coupons"`
	CouponsToRedemptions *rateJSON          `json:"coupons_to_redemptions"`
	ByBranch             []branchRateJSON   `json:"by_branch"`
	ByActivity           []activityRateJSON `json:"by_activity"`
}

// handleConversion answers GET /me/business/metrics/conversion?period=7|30|90 (Pro).
func (h *Handlers) handleConversion(w http.ResponseWriter, r *http.Request) {
	days, ok := h.periodParam(w, r)
	if !ok {
		return
	}
	got, err := h.Conversion.Handle(r.Context(), callerID(r), days)
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

	out := conversionJSON{
		sampleJSON:           sample,
		ViewsToCoupons:       toRateJSON(got.ViewsToCoupons),
		CouponsToRedemptions: toRateJSON(got.CouponsToRedemptions),
		ByBranch:             make([]branchRateJSON, 0, len(got.ByBranch)),
		ByActivity:           make([]activityRateJSON, 0, len(got.ByActivity)),
	}
	for _, b := range got.ByBranch {
		out.ByBranch = append(out.ByBranch, branchRateJSON{BranchID: b.BranchID, Name: b.Name, WhatsAppClicks: b.WhatsAppClicks, Redemptions: b.Redemptions, Rate: b.Rate})
	}
	for _, a := range got.ByActivity {
		out.ByActivity = append(out.ByActivity, activityRateJSON{ServiceID: a.ServiceID, Name: a.Name, Views: a.Views, WhatsAppClicks: a.WhatsAppClicks, Rate: a.Rate})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
