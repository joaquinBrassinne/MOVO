//go:build integration

package storage_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

// env reads the same S3_* variables the API uses; the defaults match the RustFS
// container of deploy/docker-compose.yml.
func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func localStorage(t *testing.T) *storage.S3 {
	t.Helper()

	cfg, err := config.LoadStorage(func(name string) string {
		defaults := map[string]string{
			"S3_ENDPOINT":   "http://localhost:9000",
			"S3_BUCKET":     "movo-test",
			"S3_ACCESS_KEY": "movo",
			"S3_SECRET_KEY": "movo12345",
			"S3_PATH_STYLE": "true",
		}
		return env(name, defaults[name])
	})
	if err != nil {
		t.Fatalf("LoadStorage: %v", err)
	}

	s3 := storage.NewS3(cfg)
	if err := s3.EnsureBucket(t.Context()); err != nil {
		t.Fatalf("EnsureBucket (is `make up` running?): %v", err)
	}
	return s3
}

func do(t *testing.T, method, url string, headers map[string]string, body []byte) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

// TestS3UploadAndDownloadThroughPresignedURLs uploads an object with a presigned
// PUT, reads it back with a presigned GET and checks the signature really binds
// the declared size.
func TestS3UploadAndDownloadThroughPresignedURLs(t *testing.T) {
	s3 := localStorage(t)
	payload := []byte("not really a jpeg, but the server does not care")
	key := "integration/" + time.Now().UTC().Format("20060102T150405.000000000") + ".jpg"
	t.Cleanup(func() { _ = s3.Delete(t.Context(), key) })

	up, err := s3.PresignUpload(t.Context(), storage.UploadRequest{Key: key, ContentType: "image/jpeg", Size: int64(len(payload))})
	if err != nil {
		t.Fatalf("PresignUpload: %v", err)
	}

	if res := do(t, up.Method, up.URL, up.Headers, payload[:len(payload)-1]); res.StatusCode < 400 {
		t.Errorf("upload of a body shorter than the signed size returned %d, want an error", res.StatusCode)
	}

	if res := do(t, up.Method, up.URL, up.Headers, payload); res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("upload status = %d, body %s", res.StatusCode, body)
	}

	downloadURL, err := s3.PresignDownload(t.Context(), key)
	if err != nil {
		t.Fatalf("PresignDownload: %v", err)
	}
	res := do(t, http.MethodGet, downloadURL, nil, nil)
	got, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !bytes.Equal(got, payload) {
		t.Fatalf("download = %d %q, want the uploaded bytes", res.StatusCode, got)
	}

	if err := s3.Delete(t.Context(), key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := s3.Delete(t.Context(), key); err != nil {
		t.Errorf("second Delete must be idempotent: %v", err)
	}
	if res := do(t, http.MethodGet, downloadURL, nil, nil); res.StatusCode != http.StatusNotFound {
		t.Errorf("download after delete = %d, want 404", res.StatusCode)
	}
}

func TestS3RejectsInvalidUploadsBeforeSigning(t *testing.T) {
	s3 := localStorage(t)

	if _, err := s3.PresignUpload(t.Context(), storage.UploadRequest{Key: "x.html", ContentType: "text/html", Size: 10}); err == nil {
		t.Error("an html upload must be rejected")
	}
}
