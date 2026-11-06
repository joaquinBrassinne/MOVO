package search

import "strconv"

// The business benefits the owner offers show up in the public side as a short label on
// the search cards and as the list on the profile. The search module reads them straight
// from the benefits table, so it keeps its own copy of how the label is written; the
// benefits module owns the rule and the two are pinned together by tests.

// Benefit kinds as stored.
const (
	BenefitPercent = "percent"
	BenefitAmount  = "amount"
	BenefitCustom  = "custom"
)

// ProfileBenefit is an offered benefit of the profile: active, not paused by the super
// admin, and with nothing about the business's monthly limit.
type ProfileBenefit struct {
	ID         int64
	Kind       string
	Percent    int
	AmountARS  int
	Title      string
	Conditions string
	ShortLabel string
}

// BenefitLabel is the short label of a benefit: "-15%", "-$2.000" or "Beneficio".
func BenefitLabel(kind string, percent, amountARS int) string {
	switch kind {
	case BenefitPercent:
		return "-" + strconv.Itoa(percent) + "%"
	case BenefitAmount:
		return "-$" + thousands(amountARS)
	default:
		return "Beneficio"
	}
}

// thousands writes n with a dot between groups of three digits, as pesos are written.
func thousands(n int) string {
	digits := strconv.Itoa(n)
	out := make([]byte, 0, len(digits)+len(digits)/3)
	for i := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, digits[i])
	}
	return string(out)
}
