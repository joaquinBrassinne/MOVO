package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// contactSQL reads through visible_branches, so only a public branch has a contact.
// The service is joined, not filtered: one that does not qualify leaves its columns
// NULL and the contact still comes back.
const contactSQL = `
SELECT v.business_id, v.branch_id, v.business_name, b.whatsapp_number,
       COALESCE(s.id, 0), COALESCE(s.name, '')
FROM visible_branches v
JOIN businesses b ON b.id = v.business_id
LEFT JOIN services s ON s.id = NULLIF($2::bigint, 0)
                    AND s.business_id = v.business_id
                    AND s.active
                    AND (s.branch_id IS NULL OR s.branch_id = v.branch_id)
WHERE v.branch_id = $1`

// ContactFor returns the contact of a visible branch, or search.ErrBranchNotFound.
func (r *Branches) ContactFor(ctx context.Context, branchID, serviceID int64) (search.Contact, error) {
	var c search.Contact
	err := r.pool.QueryRow(ctx, contactSQL, branchID, serviceID).Scan(
		&c.BusinessID, &c.BranchID, &c.BusinessName, &c.WhatsApp, &c.ServiceID, &c.ServiceName)
	if errors.Is(err, pgx.ErrNoRows) {
		return search.Contact{}, search.ErrBranchNotFound
	}
	if err != nil {
		return search.Contact{}, fmt.Errorf("postgres: contact of branch %d: %w", branchID, err)
	}
	return c, nil
}
