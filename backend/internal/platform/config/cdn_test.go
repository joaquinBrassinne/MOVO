package config_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

func TestLoadStorageCDNUnsetMeansNoCDN(t *testing.T) {
	t.Parallel()

	got, err := config.LoadStorage(storageEnv(nil))
	if err != nil || got.CDNURL != "" {
		t.Errorf("LoadStorage() CDNURL = %q, %v; want empty", got.CDNURL, err)
	}
}

func TestLoadStorageCDNIsNormalized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"plain", "https://cdn.example.com", "https://cdn.example.com"},
		{"trailing slash and spaces", " https://cdn.example.com/ ", "https://cdn.example.com"},
		{"many trailing slashes", "https://cdn.example.com//", "https://cdn.example.com"},
		{"path prefix kept", "https://cdn.example.com/media/", "https://cdn.example.com/media"},
		{"http for local", "http://localhost:8787", "http://localhost:8787"},
		{"explicit port", "https://cdn.example.com:8443", "https://cdn.example.com:8443"},
	}
	for _, tt := range tests {
		got, err := config.LoadStorage(storageEnv(map[string]string{"IMAGES_CDN_URL": tt.raw}))
		if err != nil {
			t.Errorf("%s: LoadStorage: %v", tt.name, err)
			continue
		}
		if got.CDNURL != tt.want {
			t.Errorf("%s: CDNURL = %q, want %q", tt.name, got.CDNURL, tt.want)
		}
	}
}

func TestLoadStorageCDNRefusesInvalidURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
	}{
		{"no scheme", "cdn.example.com"},
		{"ftp scheme", "ftp://cdn.example.com"},
		{"relative path", "/images"},
		{"no host", "https:///images"},
		{"query", "https://cdn.example.com?v=1"},
		{"bare question mark", "https://cdn.example.com/?"},
		{"fragment", "https://cdn.example.com#top"},
		{"user info", "https://user:pass@cdn.example.com"},
		{"opaque", "https:cdn.example.com"},
	}
	for _, tt := range tests {
		if _, err := config.LoadStorage(storageEnv(map[string]string{"IMAGES_CDN_URL": tt.raw})); err == nil {
			t.Errorf("%s: LoadStorage(%q) error = nil, want an error", tt.name, tt.raw)
		}
	}
}
