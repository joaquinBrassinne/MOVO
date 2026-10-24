// Package tokens signs and verifies the short-lived access tokens: HS256 JWTs
// that carry the user id and role, so the API stays stateless.
package tokens

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

const (
	issuer = "movo"
	// minSecretLen is the key size of HMAC-SHA256: a shorter secret weakens it.
	minSecretLen = 32
)

// claims are the registered claims plus the role. The user id travels as "sub".
type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// Signer issues and verifies access tokens with one shared secret. It
// implements identity.AccessTokenIssuer and httpx.TokenVerifier.
type Signer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewSigner builds a Signer. The secret must be at least 32 bytes and ttl must
// be positive. A nil now uses time.Now for verification.
func NewSigner(secret []byte, ttl time.Duration, now func() time.Time) (*Signer, error) {
	if len(secret) < minSecretLen {
		return nil, fmt.Errorf("tokens: secret must be at least %d bytes, got %d", minSecretLen, len(secret))
	}
	if ttl <= 0 {
		return nil, errors.New("tokens: ttl must be positive")
	}
	if now == nil {
		now = time.Now
	}

	return &Signer{secret: secret, ttl: ttl, now: now}, nil
}

// Issue signs a token for the user, valid from now for the configured ttl.
func (s *Signer) Issue(userID int64, role domain.Role, now time.Time) (string, time.Time, error) {
	expires := now.Add(s.ttl)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role: string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	})

	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("tokens: sign: %w", err)
	}

	return signed, expires, nil
}

// Verify checks signature, algorithm, issuer and expiry, and returns who the
// token belongs to. Only HS256 is accepted, which closes the "alg: none" and
// algorithm-confusion downgrades.
func (s *Signer) Verify(raw string) (httpx.Principal, error) {
	var parsed claims

	_, err := jwt.ParseWithClaims(raw, &parsed,
		func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil {
		return httpx.Principal{}, fmt.Errorf("tokens: verify: %w", err)
	}

	userID, err := strconv.ParseInt(parsed.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return httpx.Principal{}, errors.New("tokens: subject is not a user id")
	}

	role, err := domain.ParseRole(parsed.Role)
	if err != nil {
		return httpx.Principal{}, fmt.Errorf("tokens: %w", err)
	}

	return httpx.Principal{UserID: userID, Role: string(role)}, nil
}
