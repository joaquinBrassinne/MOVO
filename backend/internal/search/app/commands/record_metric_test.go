package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
)

type fakeCounter struct {
	events []search.Event
	days   []time.Time
	err    error
}

func (f *fakeCounter) Increment(_ context.Context, e search.Event, day time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, e)
	f.days = append(f.days, day)
	return nil
}

const chrome = "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/126.0 Mobile Safari/537.36"

var profileView = search.Event{Metric: search.ProfileView, BusinessID: 7}

func TestIsBot(t *testing.T) {
	t.Parallel()

	for ua, want := range map[string]bool{
		chrome: false,
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1": false,
		"":    true,
		"   ": true,
		"Googlebot/2.1 (+http://www.google.com/bot.html)":       true,
		"Mozilla/5.0 (compatible; bingbot/2.0)":                 true,
		"WhatsApp/2.24.8.80 A":                                  true,
		"facebookexternalhit/1.1":                               true,
		"curl/8.5.0":                                            true,
		"python-requests/2.31.0":                                true,
		"Go-http-client/2.0":                                    true,
		"Mozilla/5.0 HeadlessChrome/126.0.0.0 Safari/537.36":    true,
		"UptimeRobot/2.0":                                       true,
		"Mozilla/5.0 (Linux; Android 14) Chrome/126 Lighthouse": true,
	} {
		if got := commands.IsBot(ua); got != want {
			t.Errorf("IsBot(%q) = %t, want %t", ua, got, want)
		}
	}
}

func TestRecordMetricCountsAPersonAndSkipsABot(t *testing.T) {
	t.Parallel()

	counter := &fakeCounter{}
	cmd := commands.NewRecordMetric(counter, nil)

	counted, err := cmd.Handle(t.Context(), profileView, chrome)
	if err != nil || !counted {
		t.Fatalf("a browser: counted = %t, err = %v; want counted", counted, err)
	}
	counted, err = cmd.Handle(t.Context(), profileView, "Googlebot/2.1")
	if err != nil || counted {
		t.Errorf("a bot: counted = %t, err = %v; want skipped without an error", counted, err)
	}
	if len(counter.events) != 1 || counter.events[0] != profileView {
		t.Errorf("recorded %v, want only the browser's event", counter.events)
	}
}

// TestRecordMetricCountsOnTheArgentineDay pins that 02:00 UTC still belongs to the
// previous day in Argentina (UTC-3).
func TestRecordMetricCountsOnTheArgentineDay(t *testing.T) {
	t.Parallel()

	counter := &fakeCounter{}
	clock := func() time.Time { return time.Date(2026, 3, 2, 2, 0, 0, 0, time.UTC) }
	if _, err := commands.NewRecordMetric(counter, clock).Handle(t.Context(), profileView, chrome); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got := counter.days[0].Format(time.DateOnly); got != "2026-03-01" {
		t.Errorf("day = %s, want 2026-03-01", got)
	}
}

func TestRecordMetricRejectsAnInvalidEvent(t *testing.T) {
	t.Parallel()

	for name, e := range map[string]search.Event{
		"unknown metric":                 {Metric: "page_view", BusinessID: 7},
		"no business":                    {Metric: search.ProfileView},
		"profile view with a branch":     {Metric: search.ProfileView, BusinessID: 7, BranchID: 1},
		"click without a branch":         {Metric: search.WhatsAppClick, BusinessID: 7},
		"service view without a service": {Metric: search.ServiceView, BusinessID: 7, BranchID: 1},
		"service view without a branch":  {Metric: search.ServiceView, BusinessID: 7, ServiceID: 3},
		"negative service":               {Metric: search.WhatsAppClick, BusinessID: 7, BranchID: 1, ServiceID: -1},
	} {
		counter := &fakeCounter{}
		counted, err := commands.NewRecordMetric(counter, nil).Handle(t.Context(), e, chrome)
		if !errors.Is(err, search.ErrInvalidEvent) || counted || len(counter.events) != 0 {
			t.Errorf("%s: counted = %t, err = %v; want ErrInvalidEvent and nothing recorded", name, counted, err)
		}
	}
}

func TestRecordMetricPassesTheCounterError(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	counted, err := commands.NewRecordMetric(&fakeCounter{err: boom}, nil).Handle(t.Context(), profileView, chrome)
	if !errors.Is(err, boom) || counted {
		t.Errorf("counted = %t, err = %v; want the counter's error", counted, err)
	}
}
