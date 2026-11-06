package postgres

import (
	"strings"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// benefitColumns lists the columns of a benefit, prefixed with the table alias, in the
// order benefitDest reads them. Every query that returns a benefit uses the pair, so a
// new column is added in one place.
func benefitColumns(alias string) string {
	cols := []string{
		"id", "business_id", "type", "COALESCE(percent, 0)", "COALESCE(amount_ars, 0)",
		"COALESCE(title, '')", "COALESCE(conditions, '')", "active", "admin_paused",
	}
	for i, c := range cols {
		if alias == "" {
			continue
		}
		if strings.HasPrefix(c, "COALESCE(") {
			cols[i] = strings.Replace(c, "COALESCE(", "COALESCE("+alias+".", 1)
		} else {
			cols[i] = alias + "." + c
		}
	}
	return strings.Join(cols, ", ")
}

// couponTermsColumns are the columns of benefit_codes that hold the terms a coupon was
// issued under, in the order of the terms inside benefitColumns.
const couponTermsColumns = "benefit_type, benefit_percent, benefit_amount_ars, benefit_title, benefit_conditions"

// couponBenefitColumns is benefitColumns for a benefit seen through one of its coupons:
// the terms (type, percent, amount, title, conditions) come from the copy the coupon
// carries, so editing the benefit later does not change what the coupon offered; the id,
// the business and the on and paused flags are the benefit's own. The result is read with
// benefitDest. couponAlias names the relation that has the benefit_* columns.
func couponBenefitColumns(couponAlias, benefitAlias string) string {
	c, b := couponAlias+".", benefitAlias+"."
	return strings.Join([]string{
		b + "id", b + "business_id",
		c + "benefit_type", "COALESCE(" + c + "benefit_percent, 0)", "COALESCE(" + c + "benefit_amount_ars, 0)",
		"COALESCE(" + c + "benefit_title, '')", "COALESCE(" + c + "benefit_conditions, '')",
		b + "active", b + "admin_paused",
	}, ", ")
}

// benefitDest returns the scan targets that match benefitColumns.
func benefitDest(b *domain.Benefit) []any {
	return []any{
		&b.ID, &b.BusinessID, &b.Type, &b.Percent, &b.AmountARS,
		&b.Title, &b.Conditions, &b.Active, &b.AdminPaused,
	}
}
