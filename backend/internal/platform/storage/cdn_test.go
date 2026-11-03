package storage_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

func newTestS3(cdnURL string) *storage.S3 {
	return storage.NewS3(config.Storage{
		Endpoint:   "http://localhost:9000",
		Region:     "us-east-1",
		Bucket:     "movo",
		AccessKey:  "key",
		SecretKey:  "secret",
		PathStyle:  true,
		PresignTTL: time.Minute,
		CDNURL:     cdnURL,
	})
}

func TestPresignDownloadWithoutCDNSignsTheBucketURL(t *testing.T) {
	t.Parallel()

	got, err := newTestS3("").PresignDownload(context.Background(), "businesses/1/abc.jpg")
	if err != nil {
		t.Fatalf("PresignDownload: %v", err)
	}
	if !strings.HasPrefix(got, "http://localhost:9000/movo/businesses/1/abc.jpg?") || !strings.Contains(got, "X-Amz-Signature=") {
		t.Errorf("PresignDownload() = %q, want a presigned URL on the bucket", got)
	}
}

func TestPresignDownloadWithCDNReturnsThePublicURL(t *testing.T) {
	t.Parallel()

	got, err := newTestS3("https://cdn.example.com").PresignDownload(context.Background(), "businesses/1/abc.jpg")
	if err != nil {
		t.Fatalf("PresignDownload: %v", err)
	}
	if want := "https://cdn.example.com/businesses/1/abc.jpg"; got != want {
		t.Errorf("PresignDownload() = %q, want %q", got, want)
	}
}

func TestPresignDownloadWithCDNEscapesEachSegment(t *testing.T) {
	t.Parallel()

	got, err := newTestS3("https://cdn.example.com/media").PresignDownload(context.Background(), "businesses/1/mi foto#1.jpg")
	if err != nil {
		t.Fatalf("PresignDownload: %v", err)
	}
	if want := "https://cdn.example.com/media/businesses/1/mi%20foto%231.jpg"; got != want {
		t.Errorf("PresignDownload() = %q, want %q", got, want)
	}
}

func TestPresignDownloadWithCDNStillRefusesInvalidKeys(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"", "/abs/a.jpg", "../escape.jpg"} {
		if _, err := newTestS3("https://cdn.example.com").PresignDownload(context.Background(), key); err == nil {
			t.Errorf("PresignDownload(%q) error = nil, want an error", key)
		}
	}
}

func TestPresignUploadAlwaysGoesToTheBucketAndCachesForever(t *testing.T) {
	t.Parallel()

	got, err := newTestS3("https://cdn.example.com").PresignUpload(context.Background(), storage.UploadRequest{
		Key: "businesses/1/abc.jpg", ContentType: "image/jpeg", Size: 1024,
	})
	if err != nil {
		t.Fatalf("PresignUpload: %v", err)
	}
	if !strings.HasPrefix(got.URL, "http://localhost:9000/movo/businesses/1/abc.jpg?") {
		t.Errorf("PresignUpload URL = %q, want the bucket, not the CDN", got.URL)
	}
	if got.Headers["Cache-Control"] != storage.ImageCacheControl || got.Headers["Content-Type"] != "image/jpeg" {
		t.Errorf("PresignUpload headers = %v, want Content-Type and Cache-Control", got.Headers)
	}
}

func TestImageCacheControlIsImmutableForAYear(t *testing.T) {
	t.Parallel()

	if storage.ImageCacheControl != "public, max-age=31536000, immutable" {
		t.Errorf("ImageCacheControl = %q", storage.ImageCacheControl)
	}
}
