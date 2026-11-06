package clientip_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/clientip"
)

func trusted(cidrs ...string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		out = append(out, netip.MustParsePrefix(c))
	}
	return out
}

// resolve runs a request through the middleware and returns what a handler sees.
func resolve(t *testing.T, proxies []netip.Prefix, remoteAddr string, forwarded ...string) string {
	t.Helper()

	var got string
	handler := clientip.Middleware(proxies)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = clientip.From(r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	for _, value := range forwarded {
		req.Header.Add("X-Forwarded-For", value)
	}
	handler.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestFromIgnoresTheHeaderWhenTheConnectionIsNotATrustedProxy(t *testing.T) {
	t.Parallel()

	// No trusted proxies at all, and a trusted list that does not include the peer:
	// either way the header is a client claim and must not move the address.
	for _, proxies := range [][]netip.Prefix{nil, trusted("10.0.0.0/8")} {
		if got := resolve(t, proxies, "203.0.113.9:51000", "198.51.100.77"); got != "203.0.113.9" {
			t.Errorf("proxies %v: got %q; want the connection address 203.0.113.9", proxies, got)
		}
	}
}

func TestFromReadsTheClientBehindOneTrustedProxy(t *testing.T) {
	t.Parallel()

	got := resolve(t, trusted("10.0.0.1/32"), "10.0.0.1:4000", "198.51.100.7")
	if got != "198.51.100.7" {
		t.Errorf("got %q; want 198.51.100.7", got)
	}
}

func TestFromWalksPastTwoChainedProxies(t *testing.T) {
	t.Parallel()

	proxies := trusted("10.0.0.0/24", "192.168.1.5/32")
	// client -> https proxy (192.168.1.5) -> next (10.0.0.1) -> api
	got := resolve(t, proxies, "10.0.0.1:4000", "198.51.100.7, 192.168.1.5")
	if got != "198.51.100.7" {
		t.Errorf("got %q; want 198.51.100.7", got)
	}
}

func TestFromTakesTheFirstUntrustedAddressFromTheRight(t *testing.T) {
	t.Parallel()

	// The client wrote 1.2.3.4 itself; the proxy appended who it really saw.
	got := resolve(t, trusted("10.0.0.1/32"), "10.0.0.1:4000", "1.2.3.4, 198.51.100.7")
	if got != "198.51.100.7" {
		t.Errorf("got %q; want 198.51.100.7 and not the forged 1.2.3.4", got)
	}
}

func TestFromJoinsRepeatedHeaderLines(t *testing.T) {
	t.Parallel()

	got := resolve(t, trusted("10.0.0.0/24"), "10.0.0.1:4000", "1.2.3.4", "198.51.100.7, 10.0.0.2")
	if got != "198.51.100.7" {
		t.Errorf("got %q; want 198.51.100.7", got)
	}
}

func TestFromFallsBackToTheNearestProxyWhenTheChainIsUnusable(t *testing.T) {
	t.Parallel()

	proxies := trusted("10.0.0.0/24")
	tests := map[string][]string{
		"no header":         nil,
		"only proxies":      {"10.0.0.7, 10.0.0.8"},
		"garbage on right":  {"not-an-ip"},
		"empty entry":       {"198.51.100.7, "},
		"hostname in chain": {"198.51.100.7, proxy.internal"},
	}
	for name, header := range tests {
		got := resolve(t, proxies, "10.0.0.1:4000", header...)
		if name == "only proxies" {
			// Every hop is a proxy: the leftmost one is as far back as it goes.
			if got != "10.0.0.7" {
				t.Errorf("%s: got %q; want 10.0.0.7", name, got)
			}
			continue
		}
		if got != "10.0.0.1" {
			t.Errorf("%s: got %q; want the connection address 10.0.0.1", name, got)
		}
	}
}

func TestFromNormalisesIPv4MappedAndZones(t *testing.T) {
	t.Parallel()

	got := resolve(t, trusted("10.0.0.1/32"), "[::ffff:10.0.0.1]:4000", "::ffff:198.51.100.7")
	if got != "198.51.100.7" {
		t.Errorf("got %q; want 198.51.100.7", got)
	}
}

func TestTwoClientsBehindTheSameProxyAreToldApart(t *testing.T) {
	t.Parallel()

	proxies := trusted("10.0.0.1/32")
	a := resolve(t, proxies, "10.0.0.1:4000", "198.51.100.1")
	b := resolve(t, proxies, "10.0.0.1:4000", "198.51.100.2")
	if a == b {
		t.Errorf("both clients resolved to %q; they must differ", a)
	}
}

func TestFromWithoutTheMiddlewareUsesTheConnection(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:51000"
	req.Header.Set("X-Forwarded-For", "198.51.100.7")
	if got := clientip.From(req); got != "203.0.113.9" {
		t.Errorf("got %q; want 203.0.113.9", got)
	}

	req.RemoteAddr = "unix-socket"
	if got := clientip.From(req); got != "unix-socket" {
		t.Errorf("got %q; want the raw address when it has no port", got)
	}
}
