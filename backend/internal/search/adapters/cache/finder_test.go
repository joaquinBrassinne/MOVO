package cache_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/adapters/cache"
)

type countingFinder struct {
	calls int
	last  search.Query
	out   []search.Branch
	err   error
}

func (c *countingFinder) Nearby(_ context.Context, q search.Query) ([]search.Branch, error) {
	c.calls++
	c.last = q
	return c.out, c.err
}

type mapStore struct {
	data    map[string][]byte
	ttl     time.Duration
	getErr  error
	setErr  error
	corrupt bool
}

func newMapStore() *mapStore { return &mapStore{data: map[string][]byte{}} }

func (m *mapStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	if m.getErr != nil {
		return nil, false, m.getErr
	}
	v, ok := m.data[key]
	if ok && m.corrupt {
		return []byte("{not json"), true, nil
	}
	return v, ok, nil
}

func (m *mapStore) Incr(_ context.Context, key string) error {
	if m.setErr != nil {
		return m.setErr
	}
	n, _ := strconv.Atoi(string(m.data[key]))
	m.data[key] = []byte(strconv.Itoa(n + 1))
	return nil
}

func (m *mapStore) Set(_ context.Context, key string, v []byte, ttl time.Duration) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.data[key], m.ttl = v, ttl
	return nil
}

func newQuery() search.Query {
	return search.Query{Lat: -34.6037, Lng: -58.3816, RadiusM: 5000, Limit: 21}
}

func newFinder(next search.BranchFinder, store cache.Store) *cache.Finder {
	return cache.NewFinder(next, store, time.Minute, 0.002, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestSecondSearchInTheSameCellIsServedFromCache(t *testing.T) {
	next := &countingFinder{out: []search.Branch{{BranchID: 7, BusinessName: "Gym", DistanceM: 120.5, Featured: true}}}
	store := newMapStore()
	f := newFinder(next, store)

	first, err := f.Nearby(context.Background(), newQuery())
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	near := newQuery()
	near.Lat += 0.0003 // a few dozen meters away, same cell
	second, err := f.Nearby(context.Background(), near)
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	if next.calls != 1 {
		t.Errorf("database queried %d times, want 1", next.calls)
	}
	if len(second) != 1 || second[0] != first[0] {
		t.Errorf("cached answer %+v differs from the fresh one %+v", second, first)
	}
	if store.ttl != time.Minute {
		t.Errorf("entry stored with ttl %v, want 1m", store.ttl)
	}
}

func TestMissSearchesFromTheCellCenter(t *testing.T) {
	next := &countingFinder{}
	f := newFinder(next, newMapStore())
	q := newQuery()
	if _, err := f.Nearby(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	if d := next.last.Lat - q.Lat; d > 0.002 || d < -0.002 {
		t.Errorf("origin moved %v degrees, want less than one cell", d)
	}
	if next.last.Lat == q.Lat && next.last.Lng == q.Lng {
		t.Error("origin was not snapped to a cell center")
	}
}

func TestDifferentFiltersDoNotShareAnEntry(t *testing.T) {
	next := &countingFinder{}
	f := newFinder(next, newMapStore())
	base := newQuery()
	variants := []search.Query{base}
	for _, mutate := range []func(*search.Query){
		func(q *search.Query) { q.Category = "yoga" },
		func(q *search.Query) { q.RadiusM = 10000 },
		func(q *search.Query) { q.MinRating = 4 },
		func(q *search.Query) { q.Limit = 11 },
		func(q *search.Query) { q.Offset = 20 },
		func(q *search.Query) { q.Lat += 0.01 },
		func(q *search.Query) { q.Lng += 0.01 },
	} {
		q := base
		mutate(&q)
		variants = append(variants, q)
	}
	for _, q := range variants {
		if _, err := f.Nearby(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}
	if next.calls != len(variants) {
		t.Errorf("database queried %d times for %d distinct searches", next.calls, len(variants))
	}
}

func TestStoreFailuresFallBackToTheDatabase(t *testing.T) {
	cases := map[string]*mapStore{
		"read fails":  {data: map[string][]byte{}, getErr: errors.New("down")},
		"write fails": {data: map[string][]byte{}, setErr: errors.New("down")},
	}
	for name, store := range cases {
		t.Run(name, func(t *testing.T) {
			next := &countingFinder{out: []search.Branch{{BranchID: 1}}}
			got, err := newFinder(next, store).Nearby(context.Background(), newQuery())
			if err != nil || len(got) != 1 {
				t.Fatalf("Nearby = %v, %v; want the database answer", got, err)
			}
		})
	}
}

func TestUnreadableEntryIsRefetched(t *testing.T) {
	next := &countingFinder{out: []search.Branch{{BranchID: 1}}}
	store := newMapStore()
	f := newFinder(next, store)
	if _, err := f.Nearby(context.Background(), newQuery()); err != nil {
		t.Fatal(err)
	}
	store.corrupt = true
	got, err := f.Nearby(context.Background(), newQuery())
	if err != nil || len(got) != 1 || next.calls != 2 {
		t.Fatalf("got %v, %v after %d calls; want a refetch", got, err, next.calls)
	}
}

func TestErrorsAndInvalidQueriesAreNeverCached(t *testing.T) {
	store := newMapStore()
	next := &countingFinder{err: errors.New("boom")}
	f := newFinder(next, store)
	if _, err := f.Nearby(context.Background(), newQuery()); err == nil {
		t.Fatal("database error was swallowed")
	}
	bad := newQuery()
	bad.Lat = 99
	if _, err := f.Nearby(context.Background(), bad); err != nil && !errors.Is(err, search.ErrInvalidQuery) && next.err == nil {
		t.Fatalf("unexpected error %v", err)
	}
	if len(store.data) != 0 {
		t.Errorf("store holds %d entries, want none", len(store.data))
	}
}

func TestOriginNearThePoleStaysValid(t *testing.T) {
	next := &countingFinder{}
	q := newQuery()
	q.Lat, q.Lng = 89.9999, 179.9999
	if _, err := newFinder(next, newMapStore()).Nearby(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	if err := next.last.Validate(); err != nil {
		t.Errorf("snapped origin %v,%v is not a valid point: %v", next.last.Lat, next.last.Lng, err)
	}
}
