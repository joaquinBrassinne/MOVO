package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Environment variables understood by LoadAuth.
const (
	envJWTSecret       = "JWT_SECRET"
	envAccessTokenTTL  = "ACCESS_TOKEN_TTL"
	envRefreshTokenTTL = "REFRESH_TOKEN_TTL"

	envSuperAdminEmail    = "SUPER_ADMIN_EMAIL"
	envSuperAdminPassword = "SUPER_ADMIN_PASSWORD"
)

const (
	defaultAccessTokenTTL  = 15 * time.Minute
	defaultRefreshTokenTTL = 720 * time.Hour
	// minJWTSecretLen is the key size of HMAC-SHA256.
	minJWTSecretLen = 32
)

// Auth holds the settings of authentication. Only the API needs them, so they
// load apart from Config: the migrate tool must keep running with nothing but
// DATABASE_URL.
type Auth struct {
	// JWTSecret signs the access tokens. It is required.
	JWTSecret []byte
	// AccessTokenTTL is how long an access token stays valid.
	AccessTokenTTL time.Duration
	// RefreshTokenTTL is how long a refresh token stays valid; always longer
	// than AccessTokenTTL.
	RefreshTokenTTL time.Duration
	// SuperAdminEmail and SuperAdminPassword seed the first super admin. Both are
	// empty, or both are set.
	SuperAdminEmail    string
	SuperAdminPassword string
}

// LoadAuth builds an Auth from get, normally os.Getenv. JWT_SECRET is required
// and must be at least 32 bytes; the TTLs fall back to their defaults.
func LoadAuth(get func(string) string) (Auth, error) {
	if get == nil {
		return Auth{}, errors.New("config: environment lookup must not be nil")
	}

	secret := strings.TrimSpace(get(envJWTSecret))
	if secret == "" {
		return Auth{}, fmt.Errorf("config: %s is required and must not be empty", envJWTSecret)
	}
	if len(secret) < minJWTSecretLen {
		// The value is deliberately left out of the message: it is a secret.
		return Auth{}, fmt.Errorf("config: %s must be at least %d bytes", envJWTSecret, minJWTSecretLen)
	}

	accessTTL, err := parsePositiveDuration(envAccessTokenTTL, get(envAccessTokenTTL), defaultAccessTokenTTL)
	if err != nil {
		return Auth{}, err
	}

	refreshTTL, err := parsePositiveDuration(envRefreshTokenTTL, get(envRefreshTokenTTL), defaultRefreshTokenTTL)
	if err != nil {
		return Auth{}, err
	}
	if refreshTTL <= accessTTL {
		return Auth{}, fmt.Errorf("config: %s %s must be longer than %s %s",
			envRefreshTokenTTL, refreshTTL, envAccessTokenTTL, accessTTL)
	}

	// The password is kept as given: trimming would silently change a secret.
	adminEmail := strings.TrimSpace(get(envSuperAdminEmail))
	adminPassword := get(envSuperAdminPassword)
	if (adminEmail == "") != (adminPassword == "") {
		return Auth{}, fmt.Errorf("config: %s and %s must be set together or both left empty",
			envSuperAdminEmail, envSuperAdminPassword)
	}

	return Auth{
		JWTSecret:          []byte(secret),
		AccessTokenTTL:     accessTTL,
		RefreshTokenTTL:    refreshTTL,
		SuperAdminEmail:    adminEmail,
		SuperAdminPassword: adminPassword,
	}, nil
}

// parsePositiveDuration parses a Go duration such as 15m, or returns fallback
// when raw is empty.
func parsePositiveDuration(name, raw string, fallback time.Duration) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("config: %s %q is not a valid duration such as 15m: %w", name, raw, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("config: %s %q must be positive", name, raw)
	}

	return duration, nil
}
