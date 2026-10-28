package search_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

func TestBenefitLabelOfEachKind(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		kind            string
		percent, amount int
		want            string
	}{
		{search.BenefitPercent, 15, 0, "-15%"},
		{search.BenefitPercent, 100, 0, "-100%"},
		{search.BenefitAmount, 0, 150, "-$150"},
		{search.BenefitAmount, 0, 2000, "-$2.000"},
		{search.BenefitAmount, 0, 1234567, "-$1.234.567"},
		{search.BenefitCustom, 0, 0, "Beneficio"},
	} {
		if got := search.BenefitLabel(tc.kind, tc.percent, tc.amount); got != tc.want {
			t.Errorf("BenefitLabel(%s, %d, %d) = %q, want %q", tc.kind, tc.percent, tc.amount, got, tc.want)
		}
	}
}
