package benefitshttp

import (
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type monthJSON struct {
	Month       string `json:"month"`
	Redemptions int64  `json:"redemptions"`
}

type currentMonthJSON struct {
	Month            string  `json:"month"`
	Redemptions      int64   `json:"redemptions"`
	Progress         float64 `json:"progress"`
	RecordInProgress bool    `json:"record_in_progress"`
}

type strongSlotJSON struct {
	Slot int    `json:"slot"`
	Key  string `json:"key"`
}

type starBenefitJSON struct {
	BenefitID   int64  `json:"benefit_id"`
	Name        string `json:"name"`
	Redemptions int64  `json:"redemptions"`
}

type recordsWindowJSON struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// recordsHeaderJSON is what every answer carries: the status, the history the business has and
// the minimums it needs.
type recordsHeaderJSON struct {
	Status               string            `json:"status"`
	HistoryMonths        int               `json:"history_months"`
	MinimumHistoryMonths int               `json:"minimum_history_months"`
	MinimumBestMonth     int               `json:"minimum_best_month_redemptions"`
	Window               recordsWindowJSON `json:"window"`
	BestMonthRedemptions *int64            `json:"best_month_redemptions,omitempty"`
}

// recordsJSON adds the records once the business has enough history and a big enough best month.
type recordsJSON struct {
	recordsHeaderJSON
	BestMonth     monthJSON        `json:"best_month"`
	CurrentMonth  currentMonthJSON `json:"current_month"`
	StrongWeekday int              `json:"strong_weekday"`
	StrongSlot    strongSlotJSON   `json:"strong_slot"`
	StarBenefit   *starBenefitJSON `json:"star_benefit"`
}

// handleRecords answers GET /me/business/metrics/records (Pro). It takes no period.
func (h *Handlers) handleRecords(w http.ResponseWriter, r *http.Request) {
	got, err := h.Records.Handle(r.Context(), callerID(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}

	header := recordsHeaderJSON{
		Status: got.Status, HistoryMonths: got.HistoryMonths, MinimumHistoryMonths: got.MinHistory,
		MinimumBestMonth: got.MinBestMonth, Window: recordsWindowJSON{From: got.WindowFrom, To: got.WindowTo},
	}
	if got.Status != queries.StatusOK {
		// The best month's count is a plain number, shown so the owner sees how far from 20 it is.
		if got.BestMonth.Month != "" || got.HistoryMonths >= got.MinHistory {
			best := got.BestMonth.Redemptions
			header.BestMonthRedemptions = &best
		}
		httpx.WriteJSON(w, http.StatusOK, header)
		return
	}

	out := recordsJSON{
		recordsHeaderJSON: header,
		BestMonth:         monthJSON{Month: got.BestMonth.Month, Redemptions: got.BestMonth.Redemptions},
		CurrentMonth: currentMonthJSON{
			Month: got.CurrentMonth.Month, Redemptions: got.CurrentMonth.Redemptions,
			Progress: got.CurrentMonth.Progress, RecordInProgress: got.CurrentMonth.RecordInProgress,
		},
		StrongWeekday: got.StrongWeekday,
		StrongSlot:    strongSlotJSON{Slot: got.StrongSlot, Key: got.StrongSlotKey},
	}
	if got.StarBenefit != nil {
		out.StarBenefit = &starBenefitJSON{BenefitID: got.StarBenefit.BenefitID, Name: got.StarBenefit.Name, Redemptions: got.StarBenefit.Redemptions}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
