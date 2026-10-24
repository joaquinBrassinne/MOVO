// Package storage is the Storage port over any S3-compatible object server and
// its adapters: the S3 one and an in-memory fake for tests. Callers never learn
// which server sits behind it (see docs/adr/0002).
package storage

import (
	"context"
	"errors"
	"strings"
)

// Errors reported before anything reaches the object server.
var (
	ErrKeyInvalid            = errors.New("storage: object key is not valid")
	ErrContentTypeNotAllowed = errors.New("storage: content type is not allowed")
	ErrSizeInvalid           = errors.New("storage: size is not valid")
)

const (
	// MaxImageBytes is the largest image an owner may upload.
	MaxImageBytes = 5 << 20
	maxKeyLen     = 200
)

// allowedImageTypes is a closed list. SVG is left out on purpose: it can carry
// scripts and these images are shown to the public.
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// UploadRequest describes the object a client is about to upload. Size is part
// of the signature, so the server refuses a body of any other length.
type UploadRequest struct {
	Key         string
	ContentType string
	Size        int64
}

// PresignedUpload tells the client how to upload: send Method to URL with Headers.
type PresignedUpload struct {
	URL     string
	Method  string
	Headers map[string]string
}

// Storage creates the short-lived URLs clients use to move objects, so file
// bytes never pass through the API.
type Storage interface {
	// PresignUpload validates req and returns the URL to PUT the object to.
	PresignUpload(ctx context.Context, req UploadRequest) (PresignedUpload, error)
	// PresignDownload returns a URL to read the object.
	PresignDownload(ctx context.Context, key string) (string, error)
	// Delete removes the object. It is idempotent: a missing object is not an error.
	Delete(ctx context.Context, key string) error
}

// ValidateImageUpload checks the key, the content type and the size of an image
// upload. Every adapter calls it, so the rules do not depend on the server.
func ValidateImageUpload(req UploadRequest) error {
	if err := validateKey(req.Key); err != nil {
		return err
	}
	if !allowedImageTypes[req.ContentType] {
		return ErrContentTypeNotAllowed
	}
	if req.Size <= 0 || req.Size > MaxImageBytes {
		return ErrSizeInvalid
	}
	return nil
}

func validateKey(key string) error {
	if key == "" || len(key) > maxKeyLen || strings.HasPrefix(key, "/") || strings.Contains(key, "..") {
		return ErrKeyInvalid
	}
	return nil
}
