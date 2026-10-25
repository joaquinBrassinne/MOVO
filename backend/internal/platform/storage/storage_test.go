package storage_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

func TestValidateImageUpload(t *testing.T) {
	t.Parallel()

	ok := storage.UploadRequest{Key: "businesses/1/a.jpg", ContentType: "image/jpeg", Size: 1024}

	tests := []struct {
		name   string
		mutate func(*storage.UploadRequest)
		want   error
	}{
		{"valid jpeg", func(*storage.UploadRequest) {}, nil},
		{"valid png", func(r *storage.UploadRequest) { r.ContentType = "image/png" }, nil},
		{"valid webp", func(r *storage.UploadRequest) { r.ContentType = "image/webp" }, nil},
		{"exactly the maximum size", func(r *storage.UploadRequest) { r.Size = storage.MaxImageBytes }, nil},
		{"gif", func(r *storage.UploadRequest) { r.ContentType = "image/gif" }, storage.ErrContentTypeNotAllowed},
		{"svg can carry scripts", func(r *storage.UploadRequest) { r.ContentType = "image/svg+xml" }, storage.ErrContentTypeNotAllowed},
		{"html", func(r *storage.UploadRequest) { r.ContentType = "text/html" }, storage.ErrContentTypeNotAllowed},
		{"type with parameters", func(r *storage.UploadRequest) { r.ContentType = "image/jpeg; charset=utf-8" }, storage.ErrContentTypeNotAllowed},
		{"empty file", func(r *storage.UploadRequest) { r.Size = 0 }, storage.ErrSizeInvalid},
		{"negative size", func(r *storage.UploadRequest) { r.Size = -1 }, storage.ErrSizeInvalid},
		{"one byte over", func(r *storage.UploadRequest) { r.Size = storage.MaxImageBytes + 1 }, storage.ErrSizeInvalid},
		{"empty key", func(r *storage.UploadRequest) { r.Key = "" }, storage.ErrKeyInvalid},
		{"absolute key", func(r *storage.UploadRequest) { r.Key = "/etc/passwd" }, storage.ErrKeyInvalid},
		{"parent traversal", func(r *storage.UploadRequest) { r.Key = "businesses/../x.jpg" }, storage.ErrKeyInvalid},
		{"key too long", func(r *storage.UploadRequest) { r.Key = strings.Repeat("a", 201) }, storage.ErrKeyInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := ok
			tt.mutate(&req)
			if err := storage.ValidateImageUpload(req); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestMemoryStorage(t *testing.T) {
	t.Parallel()

	mem := storage.NewMemory()
	req := storage.UploadRequest{Key: "businesses/1/a.jpg", ContentType: "image/jpeg", Size: 10}

	if _, err := mem.PresignUpload(t.Context(), storage.UploadRequest{Key: "k", ContentType: "text/html", Size: 1}); !errors.Is(err, storage.ErrContentTypeNotAllowed) {
		t.Fatalf("invalid upload error = %v, want ErrContentTypeNotAllowed", err)
	}
	if len(mem.Uploads()) != 0 {
		t.Fatal("a rejected upload must not be recorded")
	}

	up, err := mem.PresignUpload(t.Context(), req)
	if err != nil || up.URL == "" || up.Method != "PUT" || up.Headers["Content-Type"] != "image/jpeg" {
		t.Fatalf("PresignUpload = %+v, %v", up, err)
	}
	if got := mem.Uploads(); len(got) != 1 || got[0] != req {
		t.Errorf("uploads = %+v, want the presigned request", got)
	}

	if url, err := mem.PresignDownload(t.Context(), req.Key); err != nil || !strings.Contains(url, req.Key) {
		t.Errorf("PresignDownload = %q, %v", url, err)
	}

	if err := mem.Delete(t.Context(), req.Key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := mem.Delete(t.Context(), req.Key); err != nil {
		t.Errorf("Delete must be idempotent, got %v", err)
	}
	if got := mem.Deleted(); len(got) != 2 || got[0] != req.Key {
		t.Errorf("deleted = %v", got)
	}
}

func TestMemoryExistsFollowsPutAndDelete(t *testing.T) {
	t.Parallel()

	mem := storage.NewMemory()
	key := "businesses/1/a.jpg"

	if ok, err := mem.Exists(t.Context(), key); err != nil || ok {
		t.Fatalf("before upload = %v, %v; want false", ok, err)
	}
	mem.Put(key)
	if ok, err := mem.Exists(t.Context(), key); err != nil || !ok {
		t.Fatalf("after upload = %v, %v; want true", ok, err)
	}
	if err := mem.Delete(t.Context(), key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ok, _ := mem.Exists(t.Context(), key); ok {
		t.Error("a deleted object must not exist")
	}
	if _, err := mem.Exists(t.Context(), "../x"); !errors.Is(err, storage.ErrKeyInvalid) {
		t.Errorf("invalid key error = %v, want ErrKeyInvalid", err)
	}
}
