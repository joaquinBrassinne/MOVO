package queries_test

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

type directory struct{ rows []billing.SubscriptionRow }

func (d directory) List(context.Context) ([]billing.SubscriptionRow, error) { return d.rows, nil }

func day(t *testing.T, s string) domain.Day {
	t.Helper()
	parsed, err := time.Parse(time.DateOnly, s)
	if err != nil {
		t.Fatal(err)
	}
	return domain.DayFromDate(parsed)
}

func row(t *testing.T, id int64, name, whatsApp, paidUntil string) billing.SubscriptionRow {
	t.Helper()
	sub, err := domain.NewSubscription(id, domain.PlanStarter, day(t, paidUntil))
	if err != nil {
		t.Fatal(err)
	}
	return billing.SubscriptionRow{Subscription: sub, BusinessName: name, OwnerWhatsApp: whatsApp}
}

func TestListSubscriptionsFiltersByDerivedStatusAndOrdersByDate(t *testing.T) {
	t.Parallel()

	today := day(t, "2026-11-12")
	q := queries.NewListSubscriptions(directory{rows: []billing.SubscriptionRow{
		row(t, 1, "Far", "5491100000001", "2026-12-30"),
		row(t, 2, "Soon", "5491100000002", "2026-11-15"),
		row(t, 3, "Today", "5491100000003", "2026-11-12"),
		row(t, 4, "Late", "5491100000004", "2026-11-09"),
		row(t, 5, "Gone", "5491100000005", "2026-10-01"),
	}})

	for _, tc := range []struct {
		filter string
		want   []int64
	}{
		{"", []int64{5, 4, 3, 2, 1}},
		{queries.FilterActive, []int64{3, 2, 1}},
		{queries.FilterExpiring, []int64{3, 2}},
		{queries.FilterGrace, []int64{4}},
		{queries.FilterExpired, []int64{5}},
	} {
		got, err := q.Handle(t.Context(), queries.SubscriptionFilter{Status: tc.filter}, today)
		if err != nil {
			t.Fatalf("filter %q: %v", tc.filter, err)
		}
		var ids []int64
		for _, e := range got.Entries {
			ids = append(ids, e.Subscription.BusinessID)
		}
		if len(ids) != len(tc.want) {
			t.Errorf("filter %q: ids = %v, want %v", tc.filter, ids, tc.want)
			continue
		}
		for i := range ids {
			if ids[i] != tc.want[i] {
				t.Errorf("filter %q: ids = %v, want %v", tc.filter, ids, tc.want)
				break
			}
		}
	}

	if _, err := q.Handle(t.Context(), queries.SubscriptionFilter{Status: "pending"}, today); !errors.Is(err, queries.ErrSubscriptionFilterInvalid) {
		t.Errorf("unknown filter error = %v, want ErrSubscriptionFilterInvalid", err)
	}
}

func TestListSubscriptionsReportsTheDaysLeft(t *testing.T) {
	t.Parallel()

	today := day(t, "2026-11-12")
	got, err := queries.NewListSubscriptions(directory{rows: []billing.SubscriptionRow{
		row(t, 1, "Soon", "5491100000001", "2026-11-15"),
		row(t, 2, "Late", "5491100000002", "2026-11-10"),
	}}).Handle(t.Context(), queries.SubscriptionFilter{}, today)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].DaysLeft != -2 || got.Entries[1].DaysLeft != 3 {
		t.Errorf("days left = %d, %d, want -2, 3", got.Entries[0].DaysLeft, got.Entries[1].DaysLeft)
	}
}

func proRow(t *testing.T, id int64, name, paidUntil string) billing.SubscriptionRow {
	t.Helper()
	r := row(t, id, name, "", paidUntil)
	sub, err := r.Subscription.ChangePlan(domain.PlanPro)
	if err != nil {
		t.Fatal(err)
	}
	r.Subscription = sub
	return r
}

func ids(page queries.SubscriptionPage) []int64 {
	out := []int64{}
	for _, e := range page.Entries {
		out = append(out, e.Subscription.BusinessID)
	}
	return out
}

func TestListSubscriptionsNarrowsByPlanAndBusinessName(t *testing.T) {
	t.Parallel()

	today := day(t, "2026-11-12")
	q := queries.NewListSubscriptions(directory{rows: []billing.SubscriptionRow{
		row(t, 1, "Pilates Sur", "", "2026-12-01"),
		proRow(t, 2, "Gimnasio Norte", "2026-12-02"),
		proRow(t, 3, "pilates Centro", "2026-12-03"),
	}})

	for _, tc := range []struct {
		name   string
		filter queries.SubscriptionFilter
		want   []int64
	}{
		{"plan", queries.SubscriptionFilter{Plan: "pro"}, []int64{2, 3}},
		{"name ignores case", queries.SubscriptionFilter{Query: " PILATES "}, []int64{1, 3}},
		{"plan and name", queries.SubscriptionFilter{Plan: "pro", Query: "pilates"}, []int64{3}},
		{"nothing matches", queries.SubscriptionFilter{Query: "yoga"}, []int64{}},
	} {
		got, err := q.Handle(t.Context(), tc.filter, today)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if g := ids(got); len(g) != len(tc.want) || got.Total != len(tc.want) {
			t.Errorf("%s: ids = %v total %d, want %v", tc.name, g, got.Total, tc.want)
		}
	}

	if _, err := q.Handle(t.Context(), queries.SubscriptionFilter{Plan: "gold"}, today); !errors.Is(err, domain.ErrPlanInvalid) {
		t.Errorf("unknown plan error = %v, want ErrPlanInvalid", err)
	}
}

func TestListSubscriptionsPagesAndReportsTheTotal(t *testing.T) {
	t.Parallel()

	today := day(t, "2026-11-12")
	q := queries.NewListSubscriptions(directory{rows: []billing.SubscriptionRow{
		row(t, 1, "A", "", "2026-12-01"),
		row(t, 2, "B", "", "2026-12-02"),
		row(t, 3, "C", "", "2026-12-03"),
	}})

	got, err := q.Handle(t.Context(), queries.SubscriptionFilter{Limit: 2, Offset: 1}, today)
	if err != nil {
		t.Fatal(err)
	}
	if g := ids(got); len(g) != 2 || g[0] != 2 || g[1] != 3 || got.Total != 3 {
		t.Errorf("page = %v total %d, want [2 3] of 3", g, got.Total)
	}

	past, err := q.Handle(t.Context(), queries.SubscriptionFilter{Limit: 2, Offset: 9}, today)
	if err != nil || len(past.Entries) != 0 || past.Total != 3 {
		t.Errorf("past the end = %d entries total %d err %v, want 0 of 3", len(past.Entries), past.Total, err)
	}

	for _, f := range []queries.SubscriptionFilter{{Limit: -1}, {Offset: -1}, {Limit: queries.MaxSubscriptionPage + 1}} {
		if _, err := q.Handle(t.Context(), f, today); !errors.Is(err, queries.ErrPageInvalid) {
			t.Errorf("filter %+v error = %v, want ErrPageInvalid", f, err)
		}
	}
}

func TestReminderURLOpensTheOwnersChatWithTheTextAlreadyWritten(t *testing.T) {
	t.Parallel()

	link := queries.ReminderURL("5491100000001", "Alma Flow & Yoga", day(t, "2026-11-05"))
	if !strings.HasPrefix(link, "https://wa.me/5491100000001?text=") {
		t.Fatalf("link = %q", link)
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	text := parsed.Query().Get("text")
	if !strings.Contains(text, "Alma Flow & Yoga") || !strings.Contains(text, "05/11/2026") {
		t.Errorf("text = %q, want the business name and the date", text)
	}

	if got := queries.ReminderURL("", "Studio", day(t, "2026-11-05")); got != "" {
		t.Errorf("link without a number = %q, want empty", got)
	}
}
