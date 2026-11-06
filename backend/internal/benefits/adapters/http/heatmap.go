package benefitshttp

import (
	"fmt"
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type slotJSON struct {
	Key   string `json:"key"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type heatCellJSON struct {
	Weekday     int   `json:"weekday"`
	Slot        int   `json:"slot"`
	Redemptions int64 `json:"redemptions"`
}

// heatmapJSON adds the grid once the sample is big enough.
type heatmapJSON struct {
	sampleJSON
	Slots               []slotJSON     `json:"slots"`
	Redemptions         int64          `json:"redemptions"`
	PreviousRedemptions int64          `json:"previous_redemptions"`
	Cells               []heatCellJSON `json:"cells"`
}

// handleHeatmap answers GET /me/business/metrics/heatmap?period=7|30|90 (Pro).
func (h *Handlers) handleHeatmap(w http.ResponseWriter, r *http.Request) {
	days, ok := h.periodParam(w, r)
	if !ok {
		return
	}
	got, err := h.Heatmap.Handle(r.Context(), callerID(r), days)
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

	out := heatmapJSON{
		sampleJSON: sample, Redemptions: got.Redemptions, PreviousRedemptions: got.PreviousRedemptions,
		Slots: make([]slotJSON, 0, len(queries.HeatSlots)), Cells: make([]heatCellJSON, 0, len(got.Grid)),
	}
	for _, s := range queries.HeatSlots {
		out.Slots = append(out.Slots, slotJSON{Key: s.Key, Start: fmt.Sprintf("%02d:00", s.StartHour), End: fmt.Sprintf("%02d:59", s.EndHour)})
	}
	for _, c := range got.Grid {
		out.Cells = append(out.Cells, heatCellJSON{Weekday: c.Weekday, Slot: c.Slot, Redemptions: c.Redemptions})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
