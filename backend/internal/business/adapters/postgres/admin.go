package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// Admin implements business.AdminRepository and business.AdminDirectory.
type Admin struct {
	pool *pgxpool.Pool
}

// NewAdmin builds the repository over pool.
func NewAdmin(pool *pgxpool.Pool) *Admin {
	return &Admin{pool: pool}
}

// FindByID returns the business with that id.
func (r *Admin) FindByID(ctx context.Context, id int64) (domain.Business, error) {
	return scanBusiness(r.pool.QueryRow(ctx, selectBusiness+` WHERE id = $1`, id))
}

// Apply stores the new state and appends the audit entry in one transaction. The
// UPDATE only matches while the status, pilot flag and verification are still
// those of previous, so a concurrent change makes it match nothing.
func (r *Admin) Apply(ctx context.Context, b, previous domain.Business, entry business.AuditEntry) error {
	metadata, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("postgres: encode audit metadata: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
UPDATE businesses
SET status = $2, pilot = $3, verified_at = $4, verified_by = NULLIF($5, 0), updated_at = now()
WHERE id = $1 AND status = $6 AND pilot = $7 AND (verified_at IS NOT NULL) = $8`,
		b.ID, string(b.Status), b.Pilot, b.VerifiedAt, b.VerifiedBy,
		string(previous.Status), previous.Pilot, previous.Verified())
	if err != nil {
		return fmt.Errorf("postgres: update business: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM businesses WHERE id = $1)`, b.ID).Scan(&exists); err != nil {
			return fmt.Errorf("postgres: check business: %w", err)
		}
		if !exists {
			return business.ErrBusinessNotFound
		}
		return business.ErrStatusConflict
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO audit_log (actor_id, action, entity, entity_id, metadata) VALUES ($1, $2, $3, $4, $5)`,
		entry.ActorID, entry.Action, entry.Entity, entry.EntityID, metadata); err != nil {
		return fmt.Errorf("postgres: append audit entry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}

// List returns one page of businesses, those waiting for review first and then
// the most recent. Without a status filter the deleted ones are left out.
func (r *Admin) List(ctx context.Context, filter business.AdminFilter) (business.AdminPage, error) {
	status := string(filter.Status)
	const where = `WHERE ($1 = '' AND b.status <> 'deleted') OR b.status = $1`

	var page business.AdminPage
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM businesses b `+where, status).Scan(&page.Total); err != nil {
		return business.AdminPage{}, fmt.Errorf("postgres: count businesses: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
SELECT b.id, b.owner_user_id, b.category_id, b.name, b.slug, b.description, b.whatsapp_number, b.status,
       b.pilot, b.verified_at, b.verified_by, b.payment_reference, b.created_at,
       u.email, c.name, (SELECT count(*) FROM branches br WHERE br.business_id = b.id)
FROM businesses b
JOIN users u ON u.id = b.owner_user_id
JOIN categories c ON c.id = b.category_id
`+where+`
ORDER BY (b.status = 'pending_review') DESC, b.created_at DESC, b.id DESC
LIMIT $2 OFFSET $3`, status, filter.Limit, filter.Offset)
	if err != nil {
		return business.AdminPage{}, fmt.Errorf("postgres: list businesses: %w", err)
	}
	defer rows.Close()

	page.Rows = []business.AdminRow{}
	for rows.Next() {
		var (
			row    business.AdminRow
			st     string
			by     *int64
			pay    *string
			target = []any{
				&row.Business.ID, &row.Business.OwnerID, &row.Business.CategoryID, &row.Business.Name, &row.Business.Slug,
				&row.Business.Description, &row.Business.WhatsApp, &st, &row.Business.Pilot, &row.Business.VerifiedAt,
				&by, &pay, &row.Business.CreatedAt, &row.OwnerEmail, &row.CategoryName, &row.BranchCount,
			}
		)
		if err := rows.Scan(target...); err != nil {
			return business.AdminPage{}, fmt.Errorf("postgres: scan business: %w", err)
		}
		row.Business.Status = domain.Status(st)
		if by != nil {
			row.Business.VerifiedBy = *by
		}
		if pay != nil {
			row.Business.PaymentReference = *pay
		}
		page.Rows = append(page.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return business.AdminPage{}, fmt.Errorf("postgres: list businesses: %w", err)
	}
	return page, nil
}
