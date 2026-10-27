package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const (
	// codeAlphabet has 32 symbols, so a random byte masked with 31 picks one with no
	// bias. It leaves out 0, O, 1 and I, which people confuse when they read a code.
	codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	// CodeLength is how many symbols a coupon code has.
	CodeLength = 8

	accessTokenBytes = 32
)

// NormalizeCode turns a code as a person typed it into the stored form: letters in
// capitals, without spaces or hyphens ("abcd-efgh" is "ABCDEFGH"). It reports false for
// anything that cannot be a code, so a lookup is never run with it.
func NormalizeCode(raw string) (string, bool) {
	var code strings.Builder
	for _, r := range strings.ToUpper(raw) {
		if r == ' ' || r == '-' {
			continue
		}
		if !strings.ContainsRune(codeAlphabet, r) {
			return "", false
		}
		code.WriteRune(r)
	}
	if code.Len() != CodeLength {
		return "", false
	}
	return code.String(), true
}

// NewCode returns a random coupon code drawn from r, which must be a cryptographically
// secure source (crypto/rand.Reader in production).
func NewCode(r io.Reader) (string, error) {
	raw := make([]byte, CodeLength)
	if _, err := io.ReadFull(r, raw); err != nil {
		return "", fmt.Errorf("benefits: read random bytes: %w", err)
	}
	code := make([]byte, CodeLength)
	for i, b := range raw {
		code[i] = codeAlphabet[b&31]
	}
	return string(code), nil
}

// NewAccessToken returns an unguessable token for the customer to come back to their
// coupons, and the hash that is stored instead of it. Only the hash is kept: the token
// itself is shown once, when the coupon is generated.
func NewAccessToken(r io.Reader) (token string, hash []byte, err error) {
	raw := make([]byte, accessTokenBytes)
	if _, err := io.ReadFull(r, raw); err != nil {
		return "", nil, fmt.Errorf("benefits: read random bytes: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, HashAccessToken(token), nil
}

// HashAccessToken is the stored form of a token: its SHA-256. The token is random and
// long, so a plain hash is enough and lets the database look a coupon up by it.
func HashAccessToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
