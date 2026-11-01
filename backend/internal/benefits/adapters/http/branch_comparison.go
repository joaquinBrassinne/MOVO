package benefitshttp

import (
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// countJSON is a number with the same number in the previous period and its growth.
type countJSON struct {
	Value    int64    `json:"value"`
	Previous int64    `json:"previous"`
	Change   *float64 `json:"change"`
}

func toCountJSON(c queries.Count) countJSON {
	return countJSON{Value: c.Value, Previous: c.Previous, Change: c.Change}
}

type branchComparisonRowJSON struct {
	BranchID       int64     `json:"branch_id"`
	Name           string    `json:"name"`
	Active         bool      `json:"active"`
	ServiceViews   countJSON `json:"service_views"`
	WhatsAppClicks countJSON `json:"whatsapp_clicks"`
	Redemptions    countJSON `json:"redemptions"`
}

// branchComparisonJSON adds the branches once the sample is big enough.
type branchComparisonJSON struct {
	sampleJSON
	Branches []branchComparisonRowJSON `json:"branches"`
}

// handleBranchComparison answers GET /me/business/metrics/branches?period=7|30|90 (Pro).
func (h *Handlers) handleBranchComparison(w http.ResponseWriter, r *http.Request) {
	days, ok := h.periodParam(w, r)
	if !ok {
		return
	}
	got, err := h.BranchComparison.Handle(r.Context(), callerID(r), days)
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
	out := branchComparisonJSON{sampleJSON: sample, Branches: make([]branchComparisonRowJSON, 0, len(got.Branches))}
	for _, b := range got.Branches {
		out.Branches = append(out.Branches, branchComparisonRowJSON{
			BranchID: b.BranchID, Name: b.Name, Active: b.Active,
			ServiceViews: toCountJSON(b.ServiceViews), WhatsAppClicks: toCountJSON(b.WhatsAppClicks), Redemptions: toCountJSON(b.Redemptions),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
