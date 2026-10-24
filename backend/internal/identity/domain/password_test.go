package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

func TestHashPasswordProducesAnArgon2idPHCString(t *testing.T) {
	t.Parallel()

	hash, err := domain.HashPassword(validPassword)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Errorf("hash = %q, want an argon2id PHC string with the documented parameters", hash)
	}
	if strings.Count(hash, "$") != 5 {
		t.Errorf("hash = %q, want 5 PHC separators", hash)
	}
}

func TestHashPasswordUsesAFreshSaltEachTime(t *testing.T) {
	t.Parallel()

	first, err := domain.HashPassword(validPassword)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	second, err := domain.HashPassword(validPassword)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if first == second {
		t.Error("two hashes of the same password are identical, want distinct salts")
	}
}

func TestVerifyPassword(t *testing.T) {
	t.Parallel()

	hash, err := domain.HashPassword(validPassword)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	ok, err := domain.VerifyPassword(hash, validPassword)
	if err != nil || !ok {
		t.Errorf("VerifyPassword(correct) = (%v, %v), want (true, nil)", ok, err)
	}

	ok, err = domain.VerifyPassword(hash, validPassword+"x")
	if err != nil || ok {
		t.Errorf("VerifyPassword(wrong) = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestVerifyPasswordRejectsMalformedHashes(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"empty":             "",
		"not phc":           "plaintext",
		"wrong algorithm":   "$argon2i$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
		"wrong version":     "$argon2id$v=16$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
		"zero memory":       "$argon2id$v=19$m=0,t=2,p=1$c2FsdHNhbHRzYWx0$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
		"bad parameters":    "$argon2id$v=19$m=x,t=2,p=1$c2FsdHNhbHRzYWx0$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
		"bad salt encoding": "$argon2id$v=19$m=19456,t=2,p=1$***$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
		"missing hash":      "$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0$",
	}
	for name, hash := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ok, err := domain.VerifyPassword(hash, validPassword)
			if ok || !errors.Is(err, domain.ErrHashInvalid) {
				t.Errorf("VerifyPassword(%q) = (%v, %v), want (false, %v)", hash, ok, err, domain.ErrHashInvalid)
			}
		})
	}
}
