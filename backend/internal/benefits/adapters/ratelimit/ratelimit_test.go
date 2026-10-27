package ratelimit_test

import (
	"sync"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/ratelimit"
)

const window = time.Hour

type clock struct{ now time.Time }

func (c *clock) Now() time.Time          { return c.now }
func (c *clock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func start() *clock { return &clock{now: time.Date(2028, 3, 15, 12, 0, 0, 0, time.UTC)} }

func newLimiter(c *clock, maxAttempts, maxKeys int) *ratelimit.Limiter {
	return ratelimit.New(ratelimit.Config{MaxAttempts: maxAttempts, Window: window, MaxKeys: maxKeys}, c.Now)
}

func TestAllowsTheBudgetThenRefusesUntilTheWindowEnds(t *testing.T) {
	t.Parallel()

	c := start()
	l := newLimiter(c, 3, 100)

	for i := range 3 {
		if _, ok := l.Allow("ip"); !ok {
			t.Fatalf("attempt %d refused, want the budget of 3 allowed", i+1)
		}
	}
	c.Advance(20 * time.Minute)
	retry, ok := l.Allow("ip")
	if ok || retry != 40*time.Minute {
		t.Fatalf("4th attempt = %v, %v; want refused with 40m left", retry, ok)
	}
	// A refused attempt does not extend the block.
	c.Advance(40 * time.Minute)
	if _, ok := l.Allow("ip"); !ok {
		t.Error("the key must be allowed again once its window ends")
	}
}

func TestKeysDoNotShareABudget(t *testing.T) {
	t.Parallel()

	l := newLimiter(start(), 1, 100)
	if _, ok := l.Allow("a"); !ok {
		t.Fatal("first key refused")
	}
	if _, ok := l.Allow("a"); ok {
		t.Fatal("second attempt of the same key allowed")
	}
	if _, ok := l.Allow("b"); !ok {
		t.Error("another key must have its own budget")
	}
}

func TestAFullTableAllowsNewKeysInsteadOfGrowing(t *testing.T) {
	t.Parallel()

	c := start()
	l := newLimiter(c, 1, 2)
	l.Allow("a")
	l.Allow("b")

	// Both entries are live: the third key is not tracked, so it is never blocked.
	for range 3 {
		if _, ok := l.Allow("c"); !ok {
			t.Fatal("an untracked key must be allowed")
		}
	}
	if _, ok := l.Allow("a"); ok {
		t.Error("a tracked key keeps its block")
	}

	// Once the old entries elapse, the sweep makes room and the new key is tracked.
	c.Advance(window)
	l.Allow("c")
	if _, ok := l.Allow("c"); ok {
		t.Error("after the sweep the new key must be tracked and blocked at its budget")
	}
}

func TestAllowNeverGrantsMoreThanTheBudgetUnderConcurrency(t *testing.T) {
	t.Parallel()

	const budget, callers = 10, 100
	l := newLimiter(start(), budget, 100)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := l.Allow("ip"); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != budget {
		t.Errorf("allowed %d of %d, want exactly %d", allowed, callers, budget)
	}
}
