package benefitshttp

import (
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type benefitPerformanceJSON struct {
	BenefitID    int64    `json:"benefit_id"`
	Name         string   `json:"name"`
	Generated    int64    `json:"generated"`
	Redeemed     int64    `json:"redeemed"`
	Expired      int64    `json:"expired"`
	Pending      int64    `json:"pending"`
	Rate         *float64 `json:"rate"`
	PreviousRate *float64 `json:"previous_rate"`
	Change       *float64 `json:"change"`
}

// performanceJSON adds the benefits once the sample is big enough.
type performanceJSON struct {
	sampleJSON
	Benefits []benefitPerformanceJSON `json:"benefits"`
}

// handleBenefitPerformance answers GET /me/business/metrics/benefits?period=7|30|90 (Pro).
func (h *Handlers) handleBenefitPerformance(w http.ResponseWriter, r *http.Request) {
	days, ok := h.periodParam(w, r)
	if !ok {
		return
	}
	got, err := h.Performance.Handle(r.Context(), callerID(r), days)
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

	out := performanceJSON{sampleJSON: sample, Benefits: make([]benefitPerformanceJSON, 0, len(got.Benefits))}
	for _, b := range got.Benefits {
		out.Benefits = append(out.Benefits, benefitPerformanceJSON{
			BenefitID: b.BenefitID, Name: b.Name, Generated: b.Generated, Redeemed: b.Redeemed, Expired: b.Expired,
			Pending: b.Pending, Rate: b.Rate, PreviousRate: b.Previous, Change: b.Change,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
