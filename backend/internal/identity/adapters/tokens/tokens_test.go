package tokens_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/tokens"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

var (
	secret = []byte("0123456789abcdef0123456789abcdef")
	epoch  = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
)

const ttl = 15 * time.Minute

func newSigner(t *testing.T, now time.Time) *tokens.Signer {
	t.Helper()

	signer, err := tokens.NewSigner(secret, ttl, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	return signer
}

func TestNewSignerRequiresAStrongSecretAndPositiveTTL(t *testing.T) {
	t.Parallel()

	if _, err := tokens.NewSigner(make([]byte, 31), ttl, nil); err == nil {
		t.Error("a 31-byte secret was accepted, want at least 32")
	}
	if _, err := tokens.NewSigner(secret, 0, nil); err == nil {
		t.Error("a zero TTL was accepted")
	}
	if _, err := tokens.NewSigner(secret, ttl, nil); err != nil {
		t.Errorf("a valid configuration was rejected: %v", err)
	}
}

func TestIssuedTokenVerifiesToTheSamePrincipal(t *testing.T) {
	t.Parallel()
	signer := newSigner(t, epoch)

	token, expires, err := signer.Issue(42, domain.RoleSuperAdmin, epoch)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if want := epoch.Add(ttl); !expires.Equal(want) {
		t.Errorf("expiry = %v, want %v", expires, want)
	}

	principal, err := signer.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if principal != (httpx.Principal{UserID: 42, Role: "super_admin"}) {
		t.Errorf("principal = %+v, want user 42 as super_admin", principal)
	}
}

func TestVerifyRejectsAnExpiredToken(t *testing.T) {
	t.Parallel()

	token, _, err := newSigner(t, epoch).Issue(1, domain.RoleBusinessAdmin, epoch)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := newSigner(t, epoch.Add(ttl-time.Second)).Verify(token); err != nil {
		t.Errorf("a token one second before expiry was rejected: %v", err)
	}
	if _, err := newSigner(t, epoch.Add(ttl)).Verify(token); err == nil {
		t.Error("a token at its expiry instant was accepted")
	}
}

func TestVerifyRejectsForgedAndMalformedTokens(t *testing.T) {
	t.Parallel()
	signer := newSigner(t, epoch)

	valid, _, err := signer.Issue(1, domain.RoleBusinessAdmin, epoch)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	otherKey, err := tokens.NewSigner([]byte("ffffffffffffffffffffffffffffffff"), ttl, func() time.Time { return epoch })
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	foreign, _, _ := otherKey.Issue(1, domain.RoleSuperAdmin, epoch)

	tests := map[string]string{
		"empty":                 "",
		"garbage":               "not.a.jwt",
		"signed with other key": foreign,
		"truncated signature":   valid[:len(valid)-4],
		"alg none":              unsigned(t, jwt.MapClaims{"sub": "1", "role": "super_admin", "iss": "movo", "exp": epoch.Add(time.Hour).Unix()}),
		"hs512 with right key":  signWith(t, jwt.SigningMethodHS512, jwt.MapClaims{"sub": "1", "role": "super_admin", "iss": "movo", "exp": epoch.Add(time.Hour).Unix()}),
		"no expiry":             signWith(t, jwt.SigningMethodHS256, jwt.MapClaims{"sub": "1", "role": "business_admin", "iss": "movo"}),
		"wrong issuer":          signWith(t, jwt.SigningMethodHS256, jwt.MapClaims{"sub": "1", "role": "business_admin", "iss": "evil", "exp": epoch.Add(time.Hour).Unix()}),
		"unknown role":          signWith(t, jwt.SigningMethodHS256, jwt.MapClaims{"sub": "1", "role": "root", "iss": "movo", "exp": epoch.Add(time.Hour).Unix()}),
		"non numeric subject":   signWith(t, jwt.SigningMethodHS256, jwt.MapClaims{"sub": "abc", "role": "business_admin", "iss": "movo", "exp": epoch.Add(time.Hour).Unix()}),
		"missing subject":       signWith(t, jwt.SigningMethodHS256, jwt.MapClaims{"role": "business_admin", "iss": "movo", "exp": epoch.Add(time.Hour).Unix()}),
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if principal, err := signer.Verify(token); err == nil {
				t.Errorf("Verify accepted %q as %+v, want an error", name, principal)
			}
		})
	}
}

func TestTamperingWithThePayloadBreaksTheSignature(t *testing.T) {
	t.Parallel()
	signer := newSigner(t, epoch)

	token, _, err := signer.Issue(1, domain.RoleBusinessAdmin, epoch)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	parts := strings.Split(token, ".")
	forged := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"sub":"1","role":"super_admin","iss":"movo","exp":` + "9999999999" + `}`))

	if _, err := signer.Verify(parts[0] + "." + forged + "." + parts[2]); err == nil {
		t.Error("a token with an edited payload was accepted")
	}
}

func signWith(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims) string {
	t.Helper()

	signed, err := jwt.NewWithClaims(method, claims).SignedString(secret)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

// unsigned builds an "alg: none" token by hand, the classic JWT downgrade.
func unsigned(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()

	body, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	return header + "." + base64.RawURLEncoding.EncodeToString(body) + "."
}
