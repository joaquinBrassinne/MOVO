package memory_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search/adapters/memory"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

func seen(t *testing.T, d *memory.Deduper, key string) bool {
	t.Helper()
	duplicate, err := d.Seen(t.Context(), key)
	if err != nil {
		t.Fatalf("Seen(%q): %v", key, err)
	}
	return duplicate
}

func TestDeduperIgnoresARepeatOnlyWithinTheWindow(t *testing.T) {
	t.Parallel()

	c := &clock{now: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)}
	d := memory.NewDeduper(30*time.Minute, 10, c.Now)

	if seen(t, d, "a") {
		t.Fatal("the first time is not a repeat")
	}
	if !seen(t, d, "a") {
		t.Error("the same key right away is a repeat")
	}
	if seen(t, d, "b") {
		t.Error("another key is not a repeat")
	}

	c.now = c.now.Add(29 * time.Minute)
	if !seen(t, d, "a") {
		t.Error("29 minutes later it is still a repeat")
	}
	// The repeat did not extend the window: it ends 30 minutes after the first count.
	c.now = c.now.Add(2 * time.Minute)
	if seen(t, d, "a") {
		t.Error("after the window it counts again")
	}
}

func TestDeduperStaysBoundedAndLetsNewKeysThroughWhenFull(t *testing.T) {
	t.Parallel()

	c := &clock{now: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)}
	d := memory.NewDeduper(time.Hour, 2, c.Now)

	seen(t, d, "a")
	seen(t, d, "b")
	// Full of live entries: a new key is let through and is not remembered.
	for range 2 {
		if seen(t, d, "c") {
			t.Error("a key that did not fit must not be reported as a repeat")
		}
	}
	if !seen(t, d, "a") {
		t.Error("the remembered keys are still repeats")
	}

	// Once the window is over the table makes room again.
	c.now = c.now.Add(2 * time.Hour)
	if seen(t, d, "c") {
		t.Error("c is new after the sweep")
	}
	if !seen(t, d, "c") {
		t.Error("c is remembered now that there is room")
	}
}
