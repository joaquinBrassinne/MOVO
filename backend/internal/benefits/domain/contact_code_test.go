package domain_test

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

func TestNormalizePhoneGivesOneFormForOnePerson(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"+54 9 11 1234-5678", "549111234 5678", "0054 (9) 11 1234 5678", " 5491112345678 "} {
		got, err := domain.NormalizePhone(raw)
		if err != nil || got != "5491112345678" {
			t.Errorf("NormalizePhone(%q) = %q, %v; want 5491112345678", raw, got, err)
		}
	}

	for _, raw := range []string{"", "123", "011 1234 5678", "abc5491112345678", "5491112345678x", "5491112345678901"} {
		if _, err := domain.NormalizePhone(raw); !errors.Is(err, domain.ErrPhoneInvalid) {
			t.Errorf("NormalizePhone(%q) error = %v, want ErrPhoneInvalid", raw, err)
		}
	}
}

func TestNewContact(t *testing.T) {
	t.Parallel()

	got, err := domain.NewContact("  Ana  ", "+54 9 11 1234-5678")
	if err != nil || got.Name != "Ana" || got.Phone != "5491112345678" {
		t.Fatalf("NewContact = %+v, %v; want trimmed name and normalized phone", got, err)
	}

	if _, err := domain.NewContact(" ", "5491112345678"); !errors.Is(err, domain.ErrCustomerNameInvalid) {
		t.Errorf("blank name error = %v", err)
	}
	if _, err := domain.NewContact(strings.Repeat("a", 101), "5491112345678"); !errors.Is(err, domain.ErrCustomerNameInvalid) {
		t.Errorf("long name error = %v", err)
	}
	if _, err := domain.NewContact("Ana", "12"); !errors.Is(err, domain.ErrPhoneInvalid) {
		t.Errorf("bad phone error = %v", err)
	}
}

func TestNewCodeUsesTheUnambiguousAlphabet(t *testing.T) {
	t.Parallel()

	// Every byte value must land inside the alphabet; this is the stored CHECK too.
	every := make([]byte, 256)
	for i := range every {
		every[i] = byte(i)
	}
	shape := regexp.MustCompile(`^[A-HJ-NP-Z2-9]{8}$`)
	reader := bytes.NewReader(every)
	for range 32 {
		code, err := domain.NewCode(reader)
		if err != nil || !shape.MatchString(code) {
			t.Fatalf("NewCode = %q, %v; want 8 symbols without 0, O, 1, I", code, err)
		}
	}

	if _, err := domain.NewCode(iotest.ErrReader(errors.New("no entropy"))); err == nil {
		t.Error("a failing random source must be reported, never produce a code")
	}
	if _, err := domain.NewCode(bytes.NewReader([]byte{1, 2, 3})); err == nil {
		t.Error("a short read must be reported")
	}
}

func TestAccessTokenIsLongAndOnlyItsHashIsStored(t *testing.T) {
	t.Parallel()

	token, hash, err := domain.NewAccessToken(bytes.NewReader(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}
	if len(token) != 43 || strings.ContainsAny(token, "+/=") {
		t.Errorf("token = %q, want 43 URL-safe symbols", token)
	}
	if len(hash) != 32 || bytes.Contains(hash, []byte(token)) {
		t.Errorf("hash = %x, want 32 bytes", hash)
	}
	if !bytes.Equal(hash, domain.HashAccessToken(token)) {
		t.Error("hashing the token again must give the stored hash")
	}
	if bytes.Equal(domain.HashAccessToken(token+"x"), hash) {
		t.Error("another token must hash differently")
	}
	if _, _, err := domain.NewAccessToken(iotest.ErrReader(errors.New("no entropy"))); err == nil {
		t.Error("a failing random source must be reported")
	}
}
