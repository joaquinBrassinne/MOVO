package benefitshttp

import (
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// timeToRedeemJSON adds the buckets once the sample is big enough. Each bucket is a share of
// the redemptions of the period, with the share of the previous period.
type timeToRedeemJSON struct {
	sampleJSON
	Redemptions         int64     `json:"redemptions"`
	PreviousRedemptions int64     `json:"previous_redemptions"`
	SameDay             *rateJSON `json:"same_day"`
	Within24h           *rateJSON `json:"within_24h"`
	Within48h           *rateJSON `json:"within_48h"`
}

// handleTimeToRedeem answers GET /me/business/metrics/time-to-redeem?period=7|30|90 (Pro).
func (h *Handlers) handleTimeToRedeem(w http.ResponseWriter, r *http.Request) {
	days, ok := h.periodParam(w, r)
	if !ok {
		return
	}
	got, err := h.TimeToRedeem.Handle(r.Context(), callerID(r), days)
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
	httpx.WriteJSON(w, http.StatusOK, timeToRedeemJSON{
		sampleJSON: sample, Redemptions: got.Redemptions, PreviousRedemptions: got.PreviousRedemptions,
		SameDay: toRateJSON(got.SameDay), Within24h: toRateJSON(got.Within24h), Within48h: toRateJSON(got.Within48h),
	})
}
