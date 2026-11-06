package benefitshttp

import (
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type ratingJSON struct {
	Reviews         int64    `json:"reviews"`
	PreviousReviews int64    `json:"previous_reviews"`
	Average         *float64 `json:"average"`
	PreviousAverage *float64 `json:"previous_average"`
	Change          *float64 `json:"change"`
}

// reviewsOverTimeJSON adds the rating and the review rate once the sample is big enough. They
// are numbers: no review text, name or customer is ever sent.
type reviewsOverTimeJSON struct {
	sampleJSON
	Rating     ratingJSON `json:"rating"`
	ReviewRate *rateJSON  `json:"review_rate"`
}

// handleReviewsOverTime answers GET /me/business/metrics/reviews?period=7|30|90 (Pro).
func (h *Handlers) handleReviewsOverTime(w http.ResponseWriter, r *http.Request) {
	days, ok := h.periodParam(w, r)
	if !ok {
		return
	}
	got, err := h.ReviewsOverTime.Handle(r.Context(), callerID(r), days)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	sample := sampleJSON{
		Status: got.Status, Period: toPeriodJSON(got.Period), PreviousPeriod: toPeriodJSON(got.PreviousPeriod),
		CouponsGenerated: got.CouponsGenerated, MinimumSample: got.Minimum,
	}
	if got.Status != queries.StatusOK {
		httpx.WriteJSON(w, http.StatusOK, sample)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reviewsOverTimeJSON{
		sampleJSON: sample,
		Rating: ratingJSON{
			Reviews: got.Rating.Reviews, PreviousReviews: got.Rating.PreviousReviews,
			Average: got.Rating.Average, PreviousAverage: got.Rating.Previous, Change: got.Rating.Change,
		},
		ReviewRate: toRateJSON(got.ReviewRate),
	})
}
