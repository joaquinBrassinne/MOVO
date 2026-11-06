package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// refreshTokenBytes is the entropy of a refresh token: 256 bits, well above the
// 128 bits the security requirements ask for opaque tokens.
const refreshTokenBytes = 32

// NewRefreshToken returns a random opaque token for the client and the SHA-256
// digest that is stored in its place, so a leaked database cannot be replayed.
func NewRefreshToken() (plain string, hash []byte, err error) {
	raw := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("identity: read refresh token entropy: %w", err)
	}

	plain = base64.RawURLEncoding.EncodeToString(raw)

	return plain, HashRefreshToken(plain), nil
}

// HashRefreshToken returns the digest under which a refresh token is stored and
// looked up. The token is already high entropy, so a plain SHA-256 is enough.
func HashRefreshToken(plain string) []byte {
	sum := sha256.Sum256([]byte(plain))
	return sum[:]
}
