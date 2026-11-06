package ratelimit_test

import (
	"sync"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/ratelimit"
)

const window = 15 * time.Minute

type clock struct{ now time.Time }

func (c *clock) Now() time.Time          { return c.now }
func (c *clock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func newLimiter(c *clock, maxAttempts, maxKeys int) *ratelimit.Limiter {
	return ratelimit.New(ratelimit.Config{MaxAttempts: maxAttempts, Window: window, MaxKeys: maxKeys}, c.Now)
}

func start() *clock { return &clock{now: time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)} }

func failN(l *ratelimit.Limiter, key string, n int) {
	for range n {
		l.Fail(key)
	}
}

func TestAllowsUntilTheFailureBudgetIsSpent(t *testing.T) {
	t.Parallel()
	l := newLimiter(start(), 5, 100)

	failN(l, "k", 4)
	if _, ok := l.Allow("k"); !ok {
		t.Fatal("blocked after 4 failures, want it to allow the 5th attempt")
	}

	l.Fail("k")
	if retry, ok := l.Allow("k"); ok || retry != window {
		t.Errorf("Allow after 5 failures = (%v, %v), want (%v, false)", retry, ok, window)
	}
}

func TestAttemptCountsEveryTryAndRefusesOnceTheBudgetIsSpent(t *testing.T) {
	t.Parallel()
	c := start()
	l := newLimiter(c, 3, 100)

	for i := range 3 {
		if _, ok := l.Attempt("k"); !ok {
			t.Fatalf("attempt %d refused, want the first 3 within the budget", i+1)
		}
	}
	if retry, ok := l.Attempt("k"); ok || retry != window {
		t.Errorf("4th Attempt = (%v, %v), want (%v, false)", retry, ok, window)
	}
	if _, ok := l.Attempt("other"); !ok {
		t.Error("another key must have its own budget")
	}

	c.Advance(window)
	if _, ok := l.Attempt("k"); !ok {
		t.Error("still refused once the window elapsed")
	}
}

func TestAttemptNeverPassesTheBudgetWhenManyArriveTogether(t *testing.T) {
	t.Parallel()
	l := newLimiter(start(), 5, 100)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := l.Attempt("k"); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != 5 {
		t.Errorf("allowed = %d, want exactly 5", allowed)
	}
}

func TestAttemptLetsNewKeysThroughUnrecordedOnceTheTableIsFull(t *testing.T) {
	t.Parallel()
	l := newLimiter(start(), 1, 1)

	l.Attempt("first")
	for range 3 {
		if _, ok := l.Attempt("flood"); !ok {
			t.Fatal("a key beyond MaxKeys was refused, want it let through unrecorded")
		}
	}
}

func TestRetryAfterShrinksAndTheBlockLifts(t *testing.T) {
	t.Parallel()
	c := start()
	l := newLimiter(c, 3, 100)
	failN(l, "k", 3)

	c.Advance(10 * time.Minute)
	if retry, ok := l.Allow("k"); ok || retry != 5*time.Minute {
		t.Errorf("after 10 minutes Allow = (%v, %v), want (5m, false)", retry, ok)
	}

	c.Advance(5 * time.Minute)
	if _, ok := l.Allow("k"); !ok {
		t.Error("still blocked once the window elapsed")
	}

	// A new failure starts a fresh window rather than resuming the old count.
	l.Fail("k")
	if _, ok := l.Allow("k"); !ok {
		t.Error("blocked by a stale count after the window elapsed")
	}
}

func TestTheWindowIsAnchoredAtTheFirstFailure(t *testing.T) {
	t.Parallel()
	c := start()
	l := newLimiter(c, 3, 100)

	l.Fail("k")
	c.Advance(14 * time.Minute)
	failN(l, "k", 2)

	if retry, ok := l.Allow("k"); ok || retry != time.Minute {
		t.Errorf("Allow = (%v, %v), want (1m, false): later failures must not extend the block", retry, ok)
	}
}

func TestKeysAreIndependentAndResetClearsOnlyItsKey(t *testing.T) {
	t.Parallel()
	l := newLimiter(start(), 2, 100)
	failN(l, "a", 2)
	failN(l, "b", 2)

	if _, ok := l.Allow("c"); !ok {
		t.Error("an untouched key was blocked")
	}

	l.Reset("a")
	if _, ok := l.Allow("a"); !ok {
		t.Error("key a still blocked after Reset")
	}
	if _, ok := l.Allow("b"); ok {
		t.Error("Reset(a) unblocked key b")
	}
}

func TestExpiredKeysFreeRoomForNewOnes(t *testing.T) {
	t.Parallel()
	c := start()
	l := newLimiter(c, 1, 2)

	l.Fail("a")
	l.Fail("b")
	c.Advance(window)
	l.Fail("c")

	if _, ok := l.Allow("c"); ok {
		t.Error("key c was not tracked even though expired keys should have been swept")
	}
}

func TestTrackingIsCappedAndNeverBlocksUntrackedKeys(t *testing.T) {
	t.Parallel()
	l := newLimiter(start(), 1, 2)

	l.Fail("a")
	l.Fail("b")
	l.Fail("flood") // over the cap while a and b are alive: not tracked

	if _, ok := l.Allow("flood"); !ok {
		t.Error("an untracked key was blocked")
	}
	if _, ok := l.Allow("a"); ok {
		t.Error("the cap evicted a key that was already blocked")
	}
}

func TestConcurrentFailuresAreCountedExactly(t *testing.T) {
	t.Parallel()
	l := newLimiter(start(), 50, 100)

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.Fail("k")
		}()
	}
	wg.Wait()

	if _, ok := l.Allow("k"); ok {
		t.Error("50 concurrent failures did not reach the budget of 50")
	}
}
