package commands_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/adapters/memory"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
)

func TestRecordMetricCountsTheSameVisitorOnceInTheWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	counter := &fakeCounter{}
	cmd := commands.NewRecordMetric(counter, clock).WithDeduper(memory.NewDeduper(30*time.Minute, 100, clock))

	click := search.Event{Metric: search.WhatsAppClick, BusinessID: 7, BranchID: 1}
	otherBranch := search.Event{Metric: search.WhatsAppClick, BusinessID: 7, BranchID: 2}
	service := search.Event{Metric: search.ServiceView, BusinessID: 7, BranchID: 1, ServiceID: 3}
	otherService := search.Event{Metric: search.ServiceView, BusinessID: 7, BranchID: 1, ServiceID: 4}

	for _, tc := range []struct {
		name    string
		event   search.Event
		address string
		want    bool
	}{
		{"first click", click, "198.51.100.1", true},
		{"same click, same address", click, "198.51.100.1", false},
		{"same click, other address", click, "198.51.100.2", true},
		{"other branch, same address", otherBranch, "198.51.100.1", true},
		{"first service view", service, "198.51.100.1", true},
		{"same service view", service, "198.51.100.1", false},
		{"other service, same address", otherService, "198.51.100.1", true},
		{"profile view", profileView, "198.51.100.1", true},
		{"profile view again", profileView, "198.51.100.1", false},
	} {
		got, err := cmd.Handle(search.WithClient(t.Context(), tc.address), tc.event, chrome)
		if err != nil || got != tc.want {
			t.Errorf("%s: counted = %t, err = %v; want %t", tc.name, got, err, tc.want)
		}
	}

	// Without an address there is nothing to compare, so everything counts.
	for range 2 {
		if got, err := cmd.Handle(t.Context(), profileView, chrome); err != nil || !got {
			t.Errorf("no address: counted = %t, err = %v; want counted", got, err)
		}
	}

	// After the window the same visitor counts again.
	now = now.Add(31 * time.Minute)
	if got, err := cmd.Handle(search.WithClient(t.Context(), "198.51.100.1"), click, chrome); err != nil || !got {
		t.Errorf("after the window: counted = %t, err = %v; want counted", got, err)
	}
}

func TestRecordMetricDoesNotDedupeWithoutADeduper(t *testing.T) {
	t.Parallel()

	counter := &fakeCounter{}
	cmd := commands.NewRecordMetric(counter, nil)
	ctx := search.WithClient(t.Context(), "198.51.100.1")
	for range 2 {
		if got, err := cmd.Handle(ctx, profileView, chrome); err != nil || !got {
			t.Fatalf("counted = %t, err = %v; want counted", got, err)
		}
	}
	if len(counter.events) != 2 {
		t.Errorf("recorded %d events, want 2", len(counter.events))
	}
}
