package domain_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

func TestHashAccessTokenIsTheSHA256OfTheToken(t *testing.T) {
	t.Parallel()

	// The SHA-256 of "abc", a published test vector.
	want, _ := hex.DecodeString("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
	if got := domain.HashAccessToken("abc"); !bytes.Equal(got, want) {
		t.Errorf("HashAccessToken(abc) = %x, want %x", got, want)
	}
	if bytes.Equal(domain.HashAccessToken("abc"), domain.HashAccessToken("abd")) {
		t.Error("two different tokens must not share a hash")
	}
}
