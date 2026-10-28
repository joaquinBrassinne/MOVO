package postgres

import (
	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// bestBenefitLateral joins, to each branch of a query over visible_branches (alias v), the
// best benefit its business offers right now: active and not paused by the super admin,
// a percent before an amount before a custom one, the biggest first. The columns are
// bb.type, bb.percent and bb.amount_ars, null when the business offers nothing.
const bestBenefitLateral = `
    LEFT JOIN LATERAL (
        SELECT bf.type, bf.percent, bf.amount_ars
        FROM benefits bf
        WHERE bf.business_id = v.business_id AND bf.active AND NOT bf.admin_paused
        ORDER BY CASE bf.type WHEN 'percent' THEN 0 WHEN 'amount' THEN 1 ELSE 2 END,
                 bf.percent DESC NULLS LAST, bf.amount_ars DESC NULLS LAST, bf.id
        LIMIT 1
    ) bb ON true`

// bestBenefit is the benefit columns of a branch row, as the database returns them.
type bestBenefit struct {
	kind    *string
	percent *int
	amount  *int
}

// label is the short label to show on the card, empty when the business offers nothing.
func (b bestBenefit) label() string {
	if b.kind == nil {
		return ""
	}
	percent, amount := 0, 0
	if b.percent != nil {
		percent = *b.percent
	}
	if b.amount != nil {
		amount = *b.amount
	}
	return search.BenefitLabel(*b.kind, percent, amount)
}
