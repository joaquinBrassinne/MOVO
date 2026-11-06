package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Environment variable understood by LoadSearchRate.
const envSearchRateLimit = "SEARCH_RATE_LIMIT"

// defaultSearchRateLimit is generous on purpose: opening the map is one request and every
// "Buscar en esta zona" another, so a person stays far below it; only a script reaches it.
const (
	defaultSearchRateLimit = 300
	maxSearchRateLimit     = 100_000
)

// SearchRate is the request budget of the public search for one client address.
type SearchRate struct {
	// PerMinute is how many search requests one address may make in a minute; 0 turns the
	// limit off.
	PerMinute int
}

// LoadSearchRate builds SearchRate from get, normally os.Getenv. Unset keeps the default; a
// value that is not a whole number from 0 to 100000 is refused at start-up.
func LoadSearchRate(get func(string) string) (SearchRate, error) {
	if get == nil {
		return SearchRate{}, errors.New("config: environment lookup must not be nil")
	}
	raw := strings.TrimSpace(get(envSearchRateLimit))
	if raw == "" {
		return SearchRate{PerMinute: defaultSearchRateLimit}, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 || n > maxSearchRateLimit {
		return SearchRate{}, fmt.Errorf("config: %s %q must be a whole number from 0 (off) to %d", envSearchRateLimit, raw, maxSearchRateLimit)
	}
	return SearchRate{PerMinute: n}, nil
}
