package commands_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

func TestPresignImageUploadBuildsAKeyInsideTheCallersBusiness(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	mem := storage.NewMemory()
	presign := commands.NewPresignImageUpload(businesses, mem, func() string { return "abc123" })
	mine, _ := businesses.FindByOwner(t.Context(), 7)

	got, err := presign.Handle(t.Context(), commands.PresignImageUploadInput{OwnerID: 7, ContentType: "image/png", Size: 2048})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	wantKey := "businesses/" + itoa(mine.ID) + "/abc123.png"
	if got.ObjectKey != wantKey {
		t.Errorf("object key = %q, want %q", got.ObjectKey, wantKey)
	}
	if got.Upload.Method != "PUT" || got.Upload.URL == "" {
		t.Errorf("upload = %+v", got.Upload)
	}
	if up := mem.Uploads(); len(up) != 1 || up[0].Key != wantKey || up[0].Size != 2048 || up[0].ContentType != "image/png" {
		t.Errorf("presigned request = %+v", up)
	}
}

func TestPresignImageUploadExtensionFollowsTheContentType(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	presign := commands.NewPresignImageUpload(businesses, storage.NewMemory(), func() string { return "id" })

	for contentType, ext := range map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"} {
		got, err := presign.Handle(t.Context(), commands.PresignImageUploadInput{OwnerID: 7, ContentType: contentType, Size: 1})
		if err != nil || !strings.HasSuffix(got.ObjectKey, ext) {
			t.Errorf("%s key = %q, %v; want suffix %s", contentType, got.ObjectKey, err, ext)
		}
	}
}

func TestPresignImageUploadRejections(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	mem := storage.NewMemory()
	presign := commands.NewPresignImageUpload(businesses, mem, func() string { return "id" })

	tests := []struct {
		name  string
		input commands.PresignImageUploadInput
		want  error
	}{
		{"owner without business", commands.PresignImageUploadInput{OwnerID: 99, ContentType: "image/png", Size: 1}, business.ErrBusinessNotFound},
		{"html", commands.PresignImageUploadInput{OwnerID: 7, ContentType: "text/html", Size: 1}, storage.ErrContentTypeNotAllowed},
		{"empty file", commands.PresignImageUploadInput{OwnerID: 7, ContentType: "image/png", Size: 0}, storage.ErrSizeInvalid},
		{"too large", commands.PresignImageUploadInput{OwnerID: 7, ContentType: "image/png", Size: storage.MaxImageBytes + 1}, storage.ErrSizeInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := presign.Handle(t.Context(), tt.input); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
	if len(mem.Uploads()) != 0 {
		t.Error("a rejected request must not be presigned")
	}
}
