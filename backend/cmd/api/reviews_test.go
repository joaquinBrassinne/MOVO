package main

import (
	"bytes"
	"strings"
	"testing"

	benefitsdomain "github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	reviewsdomain "github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// The reviews module hashes the access token on its own, to find the coupon without
// importing the benefits module. This is the only place both are visible: they must never
// disagree, or the token a customer was given would stop opening their coupon.
func TestReviewsAndBenefitsHashTheAccessTokenTheSameWay(t *testing.T) {
	t.Parallel()

	generated, hash, err := benefitsdomain.NewAccessToken(strings.NewReader(strings.Repeat("x", 32)))
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}
	for _, token := range []string{generated, "", "short", strings.Repeat("a", 200)} {
		if got, want := reviewsdomain.HashAccessToken(token), benefitsdomain.HashAccessToken(token); !bytes.Equal(got, want) {
			t.Errorf("hash of %q differs: reviews %x, benefits %x", token, got, want)
		}
	}
	if !bytes.Equal(reviewsdomain.HashAccessToken(generated), hash) {
		t.Error("the hash reviews computes does not match the one benefits stores with a generated token")
	}
}
