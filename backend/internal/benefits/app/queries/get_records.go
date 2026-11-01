package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// What "Tus récords" needs before it shows anything (SDD 4.3).
const (
	// RecordsMonths is how many complete months back the ranking looks.
	RecordsMonths = 12
	// RecordsMinHistoryMonths is how many complete months of history the business needs.
	RecordsMinHistoryMonths = 2
	// RecordsMinBestMonth is how many redemptions the best month needs.
	RecordsMinBestMonth = 20
)

// MonthRecord is a calendar month ("2006-01") with its redemptions.
type MonthRecord struct {
	Month       string
	Redemptions int64
}

// CurrentMonthRecord is the month in course against the best one: Progress is the redemptions
// so far over the best month's (0.65 is "vas al 65 %"), and RecordInProgress is true once they
// beat it.
type CurrentMonthRecord struct {
	Month            string
	Redemptions      int64
	Progress         float64
	RecordInProgress bool
}

// StarBenefit is the benefit with the most redemptions.
type StarBenefit struct {
	BenefitID   int64
	Name        string
	Redemptions int64
}

// RecordsResult is "Tus récords". With status insufficient_data only the history and the
// minimums are meaningful. The strong weekday (ISO, Monday 1 to Sunday 7), the strong slot and
// the star benefit come from the same 12 months as the ranking.
type RecordsResult struct {
	Status        string
	HistoryMonths int
	MinHistory    int
	MinBestMonth  int
	WindowFrom    string
	WindowTo      string
	BestMonth     MonthRecord
	CurrentMonth  CurrentMonthRecord
	StrongWeekday int
	StrongSlot    int
	StrongSlotKey string
	StarBenefit   *StarBenefit
}

// GetRecords answers "Tus récords" (SDD 6.17), a Pro metric. It takes no period: it looks at the
// 12 complete months before the current one, on Argentina's calendar.
type GetRecords struct {
	gate     ProGate
	benefits benefits.BenefitRepository
	reader   benefits.RecordsReader
	now      func() time.Time
}

// NewGetRecords wires the query. A nil clock uses time.Now.
func NewGetRecords(gate ProGate, repo benefits.BenefitRepository, reader benefits.RecordsReader, now func() time.Time) *GetRecords {
	if now == nil {
		now = time.Now
	}
	return &GetRecords{gate: gate, benefits: repo, reader: reader, now: now}
}

// Handle checks the plan first, then builds the records.
func (q *GetRecords) Handle(ctx context.Context, ownerID int64) (RecordsResult, error) {
	businessID, err := q.gate.Business(ctx, ownerID)
	if err != nil {
		return RecordsResult{}, err
	}
	now := q.now()
	monthStart, _ := domain.QuotaMonth(now)
	windowFrom := monthStart.AddDate(0, -RecordsMonths, 0)

	data, err := q.reader.Records(ctx, businessID, windowFrom, monthStart)
	if err != nil {
		return RecordsResult{}, fmt.Errorf("read records: %w", err)
	}

	result := RecordsResult{
		Status: StatusOK, MinHistory: RecordsMinHistoryMonths, MinBestMonth: RecordsMinBestMonth,
		WindowFrom: domain.DateInArgentina(windowFrom), WindowTo: domain.DateInArgentina(monthStart.AddDate(0, 0, -1)),
		HistoryMonths: completeMonthsOfHistory(data.FirstCouponAt, monthStart),
	}
	best := bestMonth(data.Months, windowFrom)
	if result.HistoryMonths < RecordsMinHistoryMonths || best.Redemptions < RecordsMinBestMonth {
		result.Status = StatusInsufficientData
		if result.HistoryMonths >= RecordsMinHistoryMonths {
			result.BestMonth = best
		}
		return result, nil
	}

	result.BestMonth = best
	result.CurrentMonth = CurrentMonthRecord{
		Month: monthStart.Format("2006-01"), Redemptions: data.CurrentMonthRedemptions,
		Progress:         round4(float64(data.CurrentMonthRedemptions) / float64(best.Redemptions)),
		RecordInProgress: data.CurrentMonthRedemptions > best.Redemptions,
	}
	result.StrongWeekday, result.StrongSlot = strongest(data.Cells)
	result.StrongSlotKey = HeatSlots[result.StrongSlot].Key

	if star, ok := q.starBenefit(ctx, businessID, data.Benefits); ok {
		result.StarBenefit = &star
	}
	return result, nil
}

// completeMonthsOfHistory counts the complete calendar months before monthStart that began no
// earlier than the first coupon of the business, looking back at most RecordsMonths.
func completeMonthsOfHistory(first *time.Time, monthStart time.Time) int {
	if first == nil {
		return 0
	}
	months := 0
	for i := 1; i <= RecordsMonths; i++ {
		if !monthStart.AddDate(0, -i, 0).Before(*first) {
			months++
		}
	}
	return months
}

// bestMonth is the month of the 12 before the current one with the most redemptions. A tie goes
// to the most recent month. A month with no redemptions is a month with 0, and never the best.
func bestMonth(months []benefits.MonthRedemptions, windowFrom time.Time) MonthRecord {
	counts := make(map[string]int64, len(months))
	for _, m := range months {
		counts[m.Month] = m.Redemptions
	}
	var best MonthRecord
	for i := 0; i < RecordsMonths; i++ {
		label := windowFrom.AddDate(0, i, 0).Format("2006-01")
		// >= walks from the oldest month to the newest, so a later month wins a tie.
		if n := counts[label]; n > 0 && n >= best.Redemptions {
			best = MonthRecord{Month: label, Redemptions: n}
		}
	}
	return best
}

// strongest returns the weekday and the slot with the most redemptions, each on its own. A tie
// goes to the earlier weekday or slot.
func strongest(cells []benefits.HeatCell) (weekday, slot int) {
	var byDay [8]int64
	var bySlot [4]int64
	for _, c := range cells {
		byDay[c.Weekday] += c.Redemptions
		bySlot[c.Slot] += c.Redemptions
	}
	weekday, slot = 1, 0
	for d := 1; d <= 7; d++ {
		if byDay[d] > byDay[weekday] {
			weekday = d
		}
	}
	for s := range bySlot {
		if bySlot[s] > bySlot[slot] {
			slot = s
		}
	}
	return weekday, slot
}

// starBenefit is the benefit with the most redemptions, named by its title or its short label. A
// tie goes to the oldest benefit. It reports false when the business has no benefits to name.
func (q *GetRecords) starBenefit(ctx context.Context, businessID int64, counts []benefits.BenefitRedemptions) (StarBenefit, bool) {
	var best *benefits.BenefitRedemptions
	for i := range counts {
		if best == nil || counts[i].Redemptions > best.Redemptions ||
			(counts[i].Redemptions == best.Redemptions && counts[i].BenefitID < best.BenefitID) {
			best = &counts[i]
		}
	}
	if best == nil {
		return StarBenefit{}, false
	}
	owned, err := q.benefits.List(ctx, businessID)
	if err != nil {
		return StarBenefit{}, false
	}
	for _, b := range owned {
		if b.ID == best.BenefitID {
			name := b.Title
			if name == "" {
				name = b.ShortLabel()
			}
			return StarBenefit{BenefitID: b.ID, Name: name, Redemptions: best.Redemptions}, true
		}
	}
	return StarBenefit{}, false
}
