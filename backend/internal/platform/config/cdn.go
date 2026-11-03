package config

import (
	"fmt"
	"net/url"
	"strings"
)

// envImagesCDNURL is the optional base URL of the CDN in front of the images bucket.
const envImagesCDNURL = "IMAGES_CDN_URL"

// loadImagesCDN reads IMAGES_CDN_URL. Empty means no CDN: image reads keep using presigned
// bucket URLs. Otherwise it must be an absolute http(s) URL without query or fragment; the
// trailing slashes are dropped so the storage adapter can append "/<key>" to it.
func loadImagesCDN(get func(string) string) (string, error) {
	raw := strings.TrimSpace(get(envImagesCDNURL))
	if raw == "" {
		return "", nil
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("config: %s %q must be an absolute http(s) URL", envImagesCDNURL, raw)
	}
	if strings.ContainsAny(raw, "?#") {
		return "", fmt.Errorf("config: %s %q must not have a query or a fragment", envImagesCDNURL, raw)
	}
	return strings.TrimRight(raw, "/"), nil
}
