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
		got, err := q.Handle(t.Context(), tc.filter, today)
		if err != nil {
			t.Fatalf("filter %q: %v", tc.filter, err)
		}
		var ids []int64
		for _, e := range got {
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

	if _, err := q.Handle(t.Context(), "pending", today); !errors.Is(err, queries.ErrSubscriptionFilterInvalid) {
		t.Errorf("unknown filter error = %v, want ErrSubscriptionFilterInvalid", err)
	}
}

func TestListSubscriptionsReportsTheDaysLeft(t *testing.T) {
	t.Parallel()

	today := day(t, "2026-11-12")
	got, err := queries.NewListSubscriptions(directory{rows: []billing.SubscriptionRow{
		row(t, 1, "Soon", "5491100000001", "2026-11-15"),
		row(t, 2, "Late", "5491100000002", "2026-11-10"),
	}}).Handle(t.Context(), "", today)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].DaysLeft != -2 || got[1].DaysLeft != 3 {
		t.Errorf("days left = %d, %d, want -2, 3", got[0].DaysLeft, got[1].DaysLeft)
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
