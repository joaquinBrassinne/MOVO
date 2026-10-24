package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

func storageEnv(overrides map[string]string) func(string) string {
	env := map[string]string{
		"S3_BUCKET":     "movo",
		"S3_ACCESS_KEY": "key",
		"S3_SECRET_KEY": "secret",
	}
	for k, v := range overrides {
		env[k] = v
	}
	return func(name string) string { return env[name] }
}

func TestLoadStorageDefaults(t *testing.T) {
	t.Parallel()

	got, err := config.LoadStorage(storageEnv(nil))
	if err != nil {
		t.Fatalf("LoadStorage: %v", err)
	}
	if got.Bucket != "movo" || got.AccessKey != "key" || got.SecretKey != "secret" {
		t.Errorf("credentials/bucket = %+v", got)
	}
	if got.Endpoint != "" || got.Region != "us-east-1" || got.PathStyle {
		t.Errorf("endpoint/region/pathStyle = %q/%q/%v, want AWS defaults", got.Endpoint, got.Region, got.PathStyle)
	}
	if got.PresignTTL != 10*time.Minute {
		t.Errorf("PresignTTL = %s, want 10m", got.PresignTTL)
	}
}

func TestLoadStorageOverrides(t *testing.T) {
	t.Parallel()

	got, err := config.LoadStorage(storageEnv(map[string]string{
		"S3_ENDPOINT":      " http://localhost:9000 ",
		"S3_REGION":        "auto",
		"S3_PATH_STYLE":    "true",
		"S3_PRESIGN_TTL":   "2m",
		"S3_ENSURE_BUCKET": "true",
	}))
	if err != nil {
		t.Fatalf("LoadStorage: %v", err)
	}
	if got.Endpoint != "http://localhost:9000" || got.Region != "auto" || !got.PathStyle || got.PresignTTL != 2*time.Minute || !got.EnsureBucket {
		t.Errorf("storage = %+v", got)
	}
}

func TestLoadStorageRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		overrides map[string]string
		wantInMsg string
	}{
		{"missing bucket", map[string]string{"S3_BUCKET": ""}, "S3_BUCKET"},
		{"missing access key", map[string]string{"S3_ACCESS_KEY": ""}, "S3_ACCESS_KEY"},
		{"missing secret key", map[string]string{"S3_SECRET_KEY": ""}, "S3_SECRET_KEY"},
		{"bad path style", map[string]string{"S3_PATH_STYLE": "maybe"}, "S3_PATH_STYLE"},
		{"bad ensure bucket", map[string]string{"S3_ENSURE_BUCKET": "sure"}, "S3_ENSURE_BUCKET"},
		{"bad ttl", map[string]string{"S3_PRESIGN_TTL": "soon"}, "S3_PRESIGN_TTL"},
		{"non-positive ttl", map[string]string{"S3_PRESIGN_TTL": "0s"}, "S3_PRESIGN_TTL"},
		{"endpoint without scheme", map[string]string{"S3_ENDPOINT": "localhost:9000"}, "S3_ENDPOINT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := config.LoadStorage(storageEnv(tt.overrides))
			if err == nil || !strings.Contains(err.Error(), tt.wantInMsg) {
				t.Fatalf("error = %v, want one naming %s", err, tt.wantInMsg)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Errorf("error %q leaks the secret", err)
			}
		})
	}

	if _, err := config.LoadStorage(nil); err == nil {
		t.Error("a nil lookup must be rejected")
	}
}
