package domain_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

func TestNormalizeCodeAcceptsWhatAPersonTypes(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"ABCDEFGH", "abcdefgh", " abcd-efgh ", "ABCD EFGH", "aBcD-eFgH"} {
		if got, ok := domain.NormalizeCode(raw); !ok || got != "ABCDEFGH" {
			t.Errorf("NormalizeCode(%q) = %q, %v; want ABCDEFGH", raw, got, ok)
		}
	}

	// Anything that cannot be a code is refused before any lookup.
	for _, raw := range []string{"", "ABCDEFG", "ABCDEFGHJ", "ABCDEFG0", "ABCDEFGO", "ABCDEFG1", "ABCDEFGI", "ABCD_EFG", "ABCDEFG%", "'; DROP"} {
		if got, ok := domain.NormalizeCode(raw); ok {
			t.Errorf("NormalizeCode(%q) = %q, true; want it refused", raw, got)
		}
	}
}
