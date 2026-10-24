package domain_test

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

func TestNewRefreshTokenIsRandomAndHashed(t *testing.T) {
	t.Parallel()

	plain, hash, err := domain.NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}

	raw, err := base64.RawURLEncoding.DecodeString(plain)
	if err != nil || len(raw) != 32 {
		t.Errorf("plain token = %q, want 32 random bytes in unpadded base64url (decoded %d bytes, err %v)",
			plain, len(raw), err)
	}
	if len(hash) != 32 {
		t.Errorf("hash length = %d, want a 32-byte SHA-256 digest", len(hash))
	}
	if !bytes.Equal(hash, domain.HashRefreshToken(plain)) {
		t.Error("returned hash differs from HashRefreshToken(plain)")
	}

	other, _, err := domain.NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	if other == plain {
		t.Error("two tokens are identical, want independent random values")
	}
}

func TestHashRefreshTokenIsDeterministicAndSensitive(t *testing.T) {
	t.Parallel()

	if !bytes.Equal(domain.HashRefreshToken("abc"), domain.HashRefreshToken("abc")) {
		t.Error("same input produced different hashes")
	}
	if bytes.Equal(domain.HashRefreshToken("abc"), domain.HashRefreshToken("abd")) {
		t.Error("different inputs produced the same hash")
	}
}
