package config_test

import (
	"net/netip"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

func lookup(raw string) func(string) string {
	return func(key string) string {
		if key == "TRUSTED_PROXIES" {
			return raw
		}
		return ""
	}
}

func TestLoadProxiesTrustsNobodyByDefault(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "   "} {
		got, err := config.LoadProxies(lookup(raw))
		if err != nil || len(got.Trusted) != 0 {
			t.Errorf("LoadProxies(%q) = %+v, %v; want no trusted proxies", raw, got, err)
		}
	}
}

func TestLoadProxiesReadsAddressesAndRanges(t *testing.T) {
	t.Parallel()

	got, err := config.LoadProxies(lookup(" 10.0.0.1, 172.16.0.0/12 ,::1,fd00::/8 "))
	if err != nil {
		t.Fatalf("LoadProxies: %v", err)
	}

	want := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.1/32"),
		netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("fd00::/8"),
	}
	if len(got.Trusted) != len(want) {
		t.Fatalf("Trusted = %v; want %v", got.Trusted, want)
	}
	for i := range want {
		if got.Trusted[i] != want[i] {
			t.Errorf("Trusted[%d] = %v; want %v", i, got.Trusted[i], want[i])
		}
	}
}

func TestLoadProxiesRefusesWhatIsNotAnAddressOrRange(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"proxy.internal", "10.0.0.1,,10.0.0.2", "10.0.0.0/33", "300.1.1.1", "10.0.0.1;10.0.0.2"} {
		if _, err := config.LoadProxies(lookup(raw)); err == nil {
			t.Errorf("LoadProxies(%q) must fail", raw)
		}
	}
	if _, err := config.LoadProxies(nil); err == nil {
		t.Error("a nil lookup must fail")
	}
}
