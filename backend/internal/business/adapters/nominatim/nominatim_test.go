package nominatim_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/adapters/nominatim"
)

const honduras = `[{"lat":"-34.5875","lon":"-58.4306","display_name":"Honduras 4820, Palermo, Buenos Aires"}]`

// fakeServer answers like Nominatim and records what it was asked.
type fakeServer struct {
	*httptest.Server
	hits      atomic.Int32
	mu        sync.Mutex
	lastQuery string
	lastAgent string
	lastPath  string
	body      string
	status    int
}

func newFakeServer(t *testing.T, body string, status int) *fakeServer {
	t.Helper()

	f := &fakeServer{body: body, status: status}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		f.mu.Lock()
		f.lastQuery, f.lastAgent, f.lastPath = r.URL.RawQuery, r.Header.Get("User-Agent"), r.URL.Path
		body, status := f.body, f.status
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(f.Close)
	return f
}

func newGeocoder(t *testing.T, server *fakeServer, interval, maxWait time.Duration) *nominatim.Geocoder {
	t.Helper()

	return nominatim.New(nominatim.Config{
		URL: server.URL, UserAgent: "MOVO-test/1.0 (test@example.com)", Interval: interval, MaxWait: maxWait,
	})
}

func TestGeocodeFindsAnAddressInArgentina(t *testing.T) {
	t.Parallel()

	server := newFakeServer(t, honduras, http.StatusOK)
	got, err := newGeocoder(t, server, time.Millisecond, time.Second).Geocode(t.Context(), "Honduras 4820, Palermo, Argentina")
	if err != nil {
		t.Fatalf("Geocode: %v", err)
	}
	if got.Latitude != -34.5875 || got.Longitude != -58.4306 || got.DisplayName != "Honduras 4820, Palermo, Buenos Aires" {
		t.Errorf("place = %+v", got)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if server.lastPath != "/search" || server.lastAgent != "MOVO-test/1.0 (test@example.com)" {
		t.Errorf("path %q, user agent %q; want /search with the configured agent", server.lastPath, server.lastAgent)
	}
	for _, want := range []string{"format=jsonv2", "limit=1", "countrycodes=ar", "q=Honduras+4820%2C+Palermo%2C+Argentina"} {
		if !contains(server.lastQuery, want) {
			t.Errorf("query %q lacks %q", server.lastQuery, want)
		}
	}
}

func TestGeocodeCachesResultsBecauseThePublicServerRequiresIt(t *testing.T) {
	t.Parallel()

	server := newFakeServer(t, honduras, http.StatusOK)
	geocoder := newGeocoder(t, server, time.Millisecond, time.Second)

	for _, query := range []string{"Honduras 4820, Palermo", "  honduras   4820,  PALERMO ", "Honduras 4820, Palermo"} {
		if _, err := geocoder.Geocode(t.Context(), query); err != nil {
			t.Fatalf("Geocode(%q): %v", query, err)
		}
	}
	if hits := server.hits.Load(); hits != 1 {
		t.Errorf("upstream requests = %d, want 1: equal queries (ignoring case and spacing) must come from the cache", hits)
	}
}

func TestGeocodeReportsAnUnknownAddressAndCachesItToo(t *testing.T) {
	t.Parallel()

	server := newFakeServer(t, `[]`, http.StatusOK)
	geocoder := newGeocoder(t, server, time.Millisecond, time.Second)

	for range 2 {
		if _, err := geocoder.Geocode(t.Context(), "Nowhere 0"); !errors.Is(err, business.ErrAddressNotFound) {
			t.Fatalf("error = %v, want ErrAddressNotFound", err)
		}
	}
	if hits := server.hits.Load(); hits != 1 {
		t.Errorf("upstream requests = %d, want 1: repeating a miss must not hit the server again", hits)
	}
}

func TestGeocodeFailuresAreNotCached(t *testing.T) {
	t.Parallel()

	server := newFakeServer(t, `oops`, http.StatusInternalServerError)
	geocoder := newGeocoder(t, server, time.Millisecond, time.Second)

	if _, err := geocoder.Geocode(t.Context(), "Honduras 4820"); !errors.Is(err, business.ErrGeocoderUnavailable) {
		t.Fatalf("error = %v, want ErrGeocoderUnavailable", err)
	}

	server.mu.Lock()
	server.body, server.status = honduras, http.StatusOK
	server.mu.Unlock()
	if _, err := geocoder.Geocode(t.Context(), "Honduras 4820"); err != nil {
		t.Errorf("after the service recovers = %v, want success: a failure must not be remembered", err)
	}
}

func TestGeocodeRejectsUnusableAnswers(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"not json":            `<html>`,
		"coordinates as text": `[{"lat":"north","lon":"-58.4","display_name":"x"}]`,
		"out of range":        `[{"lat":"134.5","lon":"-58.4","display_name":"x"}]`,
	} {
		server := newFakeServer(t, body, http.StatusOK)
		if _, err := newGeocoder(t, server, time.Millisecond, time.Second).Geocode(t.Context(), "x"); !errors.Is(err, business.ErrGeocoderUnavailable) {
			t.Errorf("%s: error = %v, want ErrGeocoderUnavailable", name, err)
		}
	}
}

// TestGeocodeSpacesRequestsByTheInterval pins the public server policy of one
// request per second, here shortened so the test is quick.
func TestGeocodeSpacesRequestsByTheInterval(t *testing.T) {
	t.Parallel()

	server := newFakeServer(t, honduras, http.StatusOK)
	const interval = 80 * time.Millisecond
	geocoder := newGeocoder(t, server, interval, 5*time.Second)

	start := time.Now()
	var wg sync.WaitGroup
	for _, query := range []string{"a", "b", "c"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := geocoder.Geocode(t.Context(), query); err != nil {
				t.Errorf("Geocode(%q): %v", query, err)
			}
		}()
	}
	wg.Wait()

	if elapsed := time.Since(start); elapsed < 2*interval {
		t.Errorf("three concurrent requests took %s, want at least two intervals (%s)", elapsed, 2*interval)
	}
}

func TestGeocodeGivesUpWhenTheQueueIsTooLong(t *testing.T) {
	t.Parallel()

	server := newFakeServer(t, honduras, http.StatusOK)
	geocoder := newGeocoder(t, server, 500*time.Millisecond, 100*time.Millisecond)

	if _, err := geocoder.Geocode(t.Context(), "first"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := geocoder.Geocode(t.Context(), "second"); !errors.Is(err, business.ErrGeocoderBusy) {
		t.Fatalf("second error = %v, want ErrGeocoderBusy: waiting 500ms exceeds the 100ms budget", err)
	}
	if hits := server.hits.Load(); hits != 1 {
		t.Errorf("upstream requests = %d, a refused request must not reach the server", hits)
	}
}

func TestGeocodeStopsWaitingWhenTheCallerGivesUp(t *testing.T) {
	t.Parallel()

	server := newFakeServer(t, honduras, http.StatusOK)
	geocoder := newGeocoder(t, server, 300*time.Millisecond, 5*time.Second)
	if _, err := geocoder.Geocode(t.Context(), "first"); err != nil {
		t.Fatalf("first: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if _, err := geocoder.Geocode(ctx, "second"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want the context deadline", err)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
