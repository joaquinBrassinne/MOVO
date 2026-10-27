package main

import (
	"testing"

	benefitsdomain "github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// The search module writes the short label of a benefit on its own, to read it without
// importing the benefits module. This is the only place both are visible: they must never
// disagree, or a card would say one thing and the coupon another.
func TestSearchAndBenefitsWriteTheSameShortLabel(t *testing.T) {
	t.Parallel()

	for _, b := range []benefitsdomain.Benefit{
		{Type: benefitsdomain.TypePercent, Percent: 1},
		{Type: benefitsdomain.TypePercent, Percent: 15},
		{Type: benefitsdomain.TypePercent, Percent: 100},
		{Type: benefitsdomain.TypeAmount, AmountARS: 150},
		{Type: benefitsdomain.TypeAmount, AmountARS: 2000},
		{Type: benefitsdomain.TypeAmount, AmountARS: 1234567},
		{Type: benefitsdomain.TypeCustom, Title: "Una clase gratis"},
	} {
		if got, want := search.BenefitLabel(string(b.Type), b.Percent, b.AmountARS), b.ShortLabel(); got != want {
			t.Errorf("%+v: search writes %q, benefits writes %q", b, got, want)
		}
	}
}
