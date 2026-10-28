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

// benefitDest returns the scan targets that match benefitColumns.
func benefitDest(b *domain.Benefit) []any {
	return []any{
		&b.ID, &b.BusinessID, &b.Type, &b.Percent, &b.AmountARS,
		&b.Title, &b.Conditions, &b.Active, &b.AdminPaused,
	}
}
