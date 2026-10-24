package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func TestLoadAuthDefaults(t *testing.T) {
	t.Parallel()

	auth, err := config.LoadAuth(env(map[string]string{"JWT_SECRET": testJWTSecret}))
	if err != nil {
		t.Fatalf("LoadAuth() error = %v, want nil", err)
	}

	if string(auth.JWTSecret) != testJWTSecret {
		t.Errorf("JWTSecret = %q, want the configured secret", auth.JWTSecret)
	}
	if auth.AccessTokenTTL != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want 15m", auth.AccessTokenTTL)
	}
	if auth.RefreshTokenTTL != 720*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want 720h", auth.RefreshTokenTTL)
	}
}

func TestLoadAuthParsesTTLs(t *testing.T) {
	t.Parallel()

	auth, err := config.LoadAuth(env(map[string]string{
		"JWT_SECRET":        testJWTSecret,
		"ACCESS_TOKEN_TTL":  " 5m ",
		"REFRESH_TOKEN_TTL": "48h",
	}))
	if err != nil {
		t.Fatalf("LoadAuth() error = %v, want nil", err)
	}

	if auth.AccessTokenTTL != 5*time.Minute || auth.RefreshTokenTTL != 48*time.Hour {
		t.Errorf("TTLs = (%v, %v), want (5m, 48h)", auth.AccessTokenTTL, auth.RefreshTokenTTL)
	}
}

func TestLoadAuthRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		values  map[string]string
		wantMsg string
	}{
		{"missing secret", map[string]string{}, "JWT_SECRET"},
		{"blank secret", map[string]string{"JWT_SECRET": "   "}, "JWT_SECRET"},
		{"short secret", map[string]string{"JWT_SECRET": strings.Repeat("a", 31)}, "at least 32"},
		{"bad access ttl", map[string]string{"JWT_SECRET": testJWTSecret, "ACCESS_TOKEN_TTL": "soon"}, "ACCESS_TOKEN_TTL"},
		{"zero access ttl", map[string]string{"JWT_SECRET": testJWTSecret, "ACCESS_TOKEN_TTL": "0s"}, "ACCESS_TOKEN_TTL"},
		{"negative refresh ttl", map[string]string{"JWT_SECRET": testJWTSecret, "REFRESH_TOKEN_TTL": "-1h"}, "REFRESH_TOKEN_TTL"},
		{
			"refresh not longer than access",
			map[string]string{"JWT_SECRET": testJWTSecret, "ACCESS_TOKEN_TTL": "2h", "REFRESH_TOKEN_TTL": "1h"},
			"REFRESH_TOKEN_TTL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := config.LoadAuth(env(tt.values))
			if err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("LoadAuth() error = %v, want it to mention %q", err, tt.wantMsg)
			}
		})
	}
}

func TestLoadAuthNeverEchoesTheSecret(t *testing.T) {
	t.Parallel()

	_, err := config.LoadAuth(env(map[string]string{"JWT_SECRET": "too-short-secret"}))
	if err == nil {
		t.Fatal("LoadAuth() accepted a short secret")
	}
	if strings.Contains(err.Error(), "too-short-secret") {
		t.Errorf("error %q contains the secret value", err)
	}
}

func TestLoadAuthRejectsNilSource(t *testing.T) {
	t.Parallel()

	if _, err := config.LoadAuth(nil); err == nil {
		t.Error("LoadAuth(nil) error = nil, want an error")
	}
}

func TestLoadAuthSuperAdminSeedIsOptionalButAllOrNothing(t *testing.T) {
	t.Parallel()

	auth, err := config.LoadAuth(env(map[string]string{"JWT_SECRET": testJWTSecret}))
	if err != nil || auth.SuperAdminEmail != "" || auth.SuperAdminPassword != "" {
		t.Fatalf("without the seed LoadAuth = (%+v, %v), want no seed and no error", auth, err)
	}

	auth, err = config.LoadAuth(env(map[string]string{
		"JWT_SECRET":           testJWTSecret,
		"SUPER_ADMIN_EMAIL":    "  root@example.com ",
		"SUPER_ADMIN_PASSWORD": "a long enough password",
	}))
	if err != nil {
		t.Fatalf("LoadAuth() error = %v, want nil", err)
	}
	if auth.SuperAdminEmail != "root@example.com" || auth.SuperAdminPassword != "a long enough password" {
		t.Errorf("seed = (%q, %q), want the trimmed email and the untouched password", auth.SuperAdminEmail, auth.SuperAdminPassword)
	}

	for name, values := range map[string]map[string]string{
		"email only":    {"SUPER_ADMIN_EMAIL": "root@example.com"},
		"password only": {"SUPER_ADMIN_PASSWORD": "a long enough password"},
	} {
		values["JWT_SECRET"] = testJWTSecret
		_, err := config.LoadAuth(env(values))
		if err == nil || !strings.Contains(err.Error(), "SUPER_ADMIN_EMAIL") {
			t.Errorf("%s: LoadAuth() error = %v, want it to ask for both variables", name, err)
		}
		if err != nil && strings.Contains(err.Error(), "a long enough password") {
			t.Errorf("%s: error %q contains the password", name, err)
		}
	}
}
