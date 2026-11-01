package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// HeatSlot is a stretch of the day of the heat map, in whole hours of Argentina's clock.
type HeatSlot struct {
	Key       string
	StartHour int
	EndHour   int // inclusive: the slot ends at EndHour:59
}

// HeatSlots are the four slots of the day. A slot is six hours, so the SQL can derive it from the hour.
var HeatSlots = []HeatSlot{
	{Key: "overnight", StartHour: 0, EndHour: 5},
	{Key: "morning", StartHour: 6, EndHour: 11},
	{Key: "afternoon", StartHour: 12, EndHour: 17},
	{Key: "evening", StartHour: 18, EndHour: 23},
}

// HeatmapResult is the redemptions of a period by weekday and slot. Grid has a cell for every
// weekday (Monday 1 to Sunday 7) and slot, in that order, zeros included. With status
// insufficient_data only CouponsGenerated and Minimum are meaningful.
type HeatmapResult struct {
	Status              string
	Period              benefits.MetricsPeriod
	Previous            benefits.MetricsPeriod
	CouponsGenerated    int64
	Minimum             int
	Redemptions         int64
	PreviousRedemptions int64
	Grid                []benefits.HeatCell
}

// GetHeatmap answers the redemption heat map (SDD 6.9), a Pro metric: on which weekdays and at
// what hours the coupons get redeemed.
type GetHeatmap struct {
	gate   ProGate
	reader benefits.HeatmapReader
	now    func() time.Time
}

// NewGetHeatmap wires the query. A nil clock uses time.Now.
func NewGetHeatmap(gate ProGate, reader benefits.HeatmapReader, now func() time.Time) *GetHeatmap {
	if now == nil {
		now = time.Now
	}
	return &GetHeatmap{gate: gate, reader: reader, now: now}
}

// Handle checks the plan first, then reads the period asked for (7, 30 or 90 days, 30 by default).
func (q *GetHeatmap) Handle(ctx context.Context, ownerID int64, days int) (HeatmapResult, error) {
	businessID, err := q.gate.Business(ctx, ownerID)
	if err != nil {
		return HeatmapResult{}, err
	}
	days, err = validDays(days)
	if err != nil {
		return HeatmapResult{}, err
	}

	current := periodOf(q.now(), days)
	previous := previousPeriod(current)
	got, err := q.reader.Heatmap(ctx, businessID, current, previous)
	if err != nil {
		return HeatmapResult{}, fmt.Errorf("read heatmap: %w", err)
	}

	result := HeatmapResult{
		Status: StatusOK, Period: current, Previous: previous,
		CouponsGenerated: got.CouponsGenerated, Minimum: benefits.MinimumSample,
	}
	if got.CouponsGenerated < benefits.MinimumSample {
		result.Status = StatusInsufficientData
		return result, nil
	}
	result.Redemptions, result.PreviousRedemptions = got.Redemptions, got.PreviousRedemptions
	result.Grid = fillGrid(got.Cells)
	return result, nil
}

// fillGrid puts every weekday and slot in order and gives the empty ones a zero.
func fillGrid(cells []benefits.HeatCell) []benefits.HeatCell {
	counts := map[[2]int]int64{}
	for _, c := range cells {
		counts[[2]int{c.Weekday, c.Slot}] += c.Redemptions
	}
	grid := make([]benefits.HeatCell, 0, 7*len(HeatSlots))
	for weekday := 1; weekday <= 7; weekday++ {
		for slot := range HeatSlots {
			grid = append(grid, benefits.HeatCell{Weekday: weekday, Slot: slot, Redemptions: counts[[2]int{weekday, slot}]})
		}
	}
	return grid
}
