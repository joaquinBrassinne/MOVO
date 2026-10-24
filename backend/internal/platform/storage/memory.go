package storage

import (
	"context"
	"sync"
)

// Memory is a Storage fake for tests. It validates like the real adapter and
// records what it was asked, but moves no bytes.
type Memory struct {
	mu      sync.Mutex
	uploads []UploadRequest
	deleted []string
}

// NewMemory builds an empty fake.
func NewMemory() *Memory { return &Memory{} }

// PresignUpload validates and records the request.
func (m *Memory) PresignUpload(_ context.Context, req UploadRequest) (PresignedUpload, error) {
	if err := ValidateImageUpload(req); err != nil {
		return PresignedUpload{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.uploads = append(m.uploads, req)
	return PresignedUpload{
		URL:     "https://storage.test/" + req.Key + "?signature=fake",
		Method:  "PUT",
		Headers: map[string]string{"Content-Type": req.ContentType},
	}, nil
}

// PresignDownload returns a fake URL for the key.
func (m *Memory) PresignDownload(_ context.Context, key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	return "https://storage.test/" + key + "?signature=fake", nil
}

// Delete records the deletion.
func (m *Memory) Delete(_ context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleted = append(m.deleted, key)
	return nil
}

// Uploads returns the presigned upload requests so far.
func (m *Memory) Uploads() []UploadRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]UploadRequest(nil), m.uploads...)
}

// Deleted returns the keys deleted so far, including repeats.
func (m *Memory) Deleted() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.deleted...)
}
