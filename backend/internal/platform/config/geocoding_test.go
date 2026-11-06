package config_test

import (
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

func geocodingEnv(overrides map[string]string) func(string) string {
	env := map[string]string{"GEOCODER_USER_AGENT": "MOVO/1.0 (contacto@movo.example)"}
	for k, v := range overrides {
		env[k] = v
	}
	return func(name string) string { return env[name] }
}

func TestLoadGeocoding(t *testing.T) {
	t.Parallel()

	got, err := config.LoadGeocoding(geocodingEnv(nil))
	if err != nil {
		t.Fatalf("LoadGeocoding: %v", err)
	}
	if got.URL != "https://nominatim.openstreetmap.org" || got.UserAgent != "MOVO/1.0 (contacto@movo.example)" {
		t.Errorf("geocoding = %+v, want the public Nominatim by default", got)
	}

	custom, err := config.LoadGeocoding(geocodingEnv(map[string]string{"GEOCODER_URL": " http://localhost:8088/ "}))
	if err != nil || custom.URL != "http://localhost:8088" {
		t.Errorf("custom = %+v, %v; want the URL trimmed of spaces and trailing slash", custom, err)
	}
}

func TestLoadGeocodingRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		overrides map[string]string
		wantInMsg string
	}{
		{"missing user agent", map[string]string{"GEOCODER_USER_AGENT": ""}, "GEOCODER_USER_AGENT"},
		{"blank user agent", map[string]string{"GEOCODER_USER_AGENT": "   "}, "GEOCODER_USER_AGENT"},
		{"url without scheme", map[string]string{"GEOCODER_URL": "localhost:8088"}, "GEOCODER_URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := config.LoadGeocoding(geocodingEnv(tt.overrides)); err == nil || !strings.Contains(err.Error(), tt.wantInMsg) {
				t.Fatalf("error = %v, want one naming %s", err, tt.wantInMsg)
			}
		})
	}
	if _, err := config.LoadGeocoding(nil); err == nil {
		t.Error("a nil lookup must be rejected")
	}
}
