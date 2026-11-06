package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Environment variables understood by LoadGeocoding.
const (
	envGeocoderURL       = "GEOCODER_URL"
	envGeocoderUserAgent = "GEOCODER_USER_AGENT"
)

const defaultGeocoderURL = "https://nominatim.openstreetmap.org"

// Geocoding holds the settings of the address geocoder (SDD §12.3).
type Geocoding struct {
	// URL is the Nominatim-compatible server, without trailing slash. The public
	// instance by default.
	URL string
	// UserAgent identifies MOVO to the server. The public Nominatim policy requires
	// one that names the application and a way to contact its operator, so it has
	// no default.
	UserAgent string
}

// LoadGeocoding builds a Geocoding from get, normally os.Getenv.
func LoadGeocoding(get func(string) string) (Geocoding, error) {
	if get == nil {
		return Geocoding{}, errors.New("config: environment lookup must not be nil")
	}

	userAgent := strings.TrimSpace(get(envGeocoderUserAgent))
	if userAgent == "" {
		return Geocoding{}, fmt.Errorf("config: %s is required and must name the application and a contact", envGeocoderUserAgent)
	}

	endpoint := strings.TrimRight(strings.TrimSpace(get(envGeocoderURL)), "/")
	if endpoint == "" {
		endpoint = defaultGeocoderURL
	}
	if parsed, err := url.Parse(endpoint); err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return Geocoding{}, fmt.Errorf("config: %s %q must be an http(s) URL", envGeocoderURL, endpoint)
	}

	return Geocoding{URL: endpoint, UserAgent: userAgent}, nil
}
