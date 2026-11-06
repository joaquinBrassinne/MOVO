package config

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

// Environment variable understood by LoadProxies.
const envTrustedProxies = "TRUSTED_PROXIES"

// Proxies holds the reverse proxies whose X-Forwarded-For header the API believes.
type Proxies struct {
	// Trusted are the addresses and ranges of those proxies; empty means the header is
	// never read and the client is the TCP peer.
	Trusted []netip.Prefix
}

// LoadProxies builds Proxies from get, normally os.Getenv. TRUSTED_PROXIES is a comma
// separated list of IP addresses or CIDR ranges. Unset keeps the list empty; any entry
// that is neither is refused at start-up so a typo cannot silently drop a proxy.
func LoadProxies(get func(string) string) (Proxies, error) {
	if get == nil {
		return Proxies{}, errors.New("config: environment lookup must not be nil")
	}

	raw := strings.TrimSpace(get(envTrustedProxies))
	if raw == "" {
		return Proxies{}, nil
	}

	var trusted []netip.Prefix
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		prefix, err := parseProxy(entry)
		if err != nil {
			return Proxies{}, fmt.Errorf("config: %s entry %q must be an IP address or a CIDR range", envTrustedProxies, entry)
		}
		trusted = append(trusted, prefix)
	}
	return Proxies{Trusted: trusted}, nil
}

func parseProxy(entry string) (netip.Prefix, error) {
	if strings.Contains(entry, "/") {
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return netip.Prefix{}, err
		}
		return prefix.Masked(), nil
	}

	addr, err := netip.ParseAddr(entry)
	if err != nil {
		return netip.Prefix{}, err
	}
	addr = addr.Unmap().WithZone("")
	return netip.PrefixFrom(addr, addr.BitLen()), nil
}
