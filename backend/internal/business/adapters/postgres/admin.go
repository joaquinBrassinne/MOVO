package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
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

const adminSelect = `
SELECT b.id, b.owner_user_id, b.category_id, b.name, b.slug, b.description, b.whatsapp_number, b.status,
       b.pilot, b.verified_at, b.verified_by, b.payment_reference, b.created_at,
       u.email, c.name, (SELECT count(*) FROM branches br WHERE br.business_id = b.id)
FROM businesses b
JOIN users u ON u.id = b.owner_user_id
JOIN categories c ON c.id = b.category_id`

// scanner is what pgx.Row and pgx.Rows share.
type scanner interface {
	Scan(dest ...any) error
}

func scanAdminRow(row scanner) (business.AdminRow, error) {
	var (
		r   business.AdminRow
		st  string
		by  *int64
		pay *string
	)
	err := row.Scan(
		&r.Business.ID, &r.Business.OwnerID, &r.Business.CategoryID, &r.Business.Name, &r.Business.Slug,
		&r.Business.Description, &r.Business.WhatsApp, &st, &r.Business.Pilot, &r.Business.VerifiedAt,
		&by, &pay, &r.Business.CreatedAt, &r.OwnerEmail, &r.CategoryName, &r.BranchCount,
	)
	if err != nil {
		return business.AdminRow{}, err
	}

	r.Business.Status = domain.Status(st)
	if by != nil {
		r.Business.VerifiedBy = *by
	}
	if pay != nil {
		r.Business.PaymentReference = *pay
	}
	return r, nil
}

// Find returns one business as a list row.
func (r *Admin) Find(ctx context.Context, id int64) (business.AdminRow, error) {
	row, err := scanAdminRow(r.pool.QueryRow(ctx, adminSelect+` WHERE b.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return business.AdminRow{}, business.ErrBusinessNotFound
	}
	if err != nil {
		return business.AdminRow{}, fmt.Errorf("postgres: select business row: %w", err)
	}
	return row, nil
}

// List returns one page of businesses, those waiting for review first and then the most
// recent. Without a status filter the deleted ones are left out. The search matches the
// name, slug, owner email and branch addresses, case-insensitively and with the
// wildcards of the text escaped, or an exact business id.
func (r *Admin) List(ctx context.Context, filter business.AdminFilter) (business.AdminPage, error) {
	var (
		conditions []string
		args       []any
	)
	arg := func(value any) string {
		args = append(args, value)
		return "$" + strconv.Itoa(len(args))
	}

	if filter.Status == "" {
		conditions = append(conditions, "b.status <> 'deleted'")
	} else {
		conditions = append(conditions, "b.status = "+arg(string(filter.Status)))
	}
	if filter.CategoryID != 0 {
		conditions = append(conditions, "b.category_id = "+arg(filter.CategoryID))
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		pattern, exact := arg("%"+escapeLike(q)+"%"), arg(q)
		conditions = append(conditions, "(b.name ILIKE "+pattern+" OR b.slug ILIKE "+pattern+" OR u.email ILIKE "+pattern+
			" OR EXISTS (SELECT 1 FROM branches s WHERE s.business_id = b.id AND s.address ILIKE "+pattern+")"+
			" OR b.id::text = "+exact+")")
	}
	where := " WHERE " + strings.Join(conditions, " AND ")

	var page business.AdminPage
	err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM businesses b JOIN users u ON u.id = b.owner_user_id"+where, args...).Scan(&page.Total)
	if err != nil {
		return business.AdminPage{}, fmt.Errorf("postgres: count businesses: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		adminSelect+where+
			" ORDER BY (b.status = 'pending_review') DESC, b.created_at DESC, b.id DESC LIMIT "+arg(filter.Limit)+" OFFSET "+arg(filter.Offset),
		args...)
	if err != nil {
		return business.AdminPage{}, fmt.Errorf("postgres: list businesses: %w", err)
	}
	defer rows.Close()

	page.Rows = []business.AdminRow{}
	for rows.Next() {
		row, err := scanAdminRow(rows)
		if err != nil {
			return business.AdminPage{}, fmt.Errorf("postgres: scan business: %w", err)
		}
		page.Rows = append(page.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return business.AdminPage{}, fmt.Errorf("postgres: list businesses: %w", err)
	}

	if page.Counts, err = r.counts(ctx); err != nil {
		return business.AdminPage{}, err
	}
	return page, nil
}

// counts is how many businesses each status has, for the tabs of the list.
func (r *Admin) counts(ctx context.Context) (map[domain.Status]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT status, count(*) FROM businesses GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("postgres: count by status: %w", err)
	}
	defer rows.Close()

	counts := map[domain.Status]int{}
	for rows.Next() {
		var (
			status string
			n      int
		)
		if err := rows.Scan(&status, &n); err != nil {
			return nil, fmt.Errorf("postgres: scan status count: %w", err)
		}
		counts[domain.Status(status)] = n
	}
	return counts, rows.Err()
}

// History returns the audit entries of the business, newest first.
func (r *Admin) History(ctx context.Context, id int64) ([]business.AuditRecord, error) {
	rows, err := r.pool.Query(ctx, `
SELECT u.email, a.action, a.created_at, a.metadata
FROM audit_log a
JOIN users u ON u.id = a.actor_id
WHERE a.entity = 'business' AND a.entity_id = $1
ORDER BY a.created_at DESC, a.id DESC`, id)
	if err != nil {
		return nil, fmt.Errorf("postgres: list audit history: %w", err)
	}
	defer rows.Close()

	records := []business.AuditRecord{}
	for rows.Next() {
		var (
			rec business.AuditRecord
			raw []byte
		)
		if err := rows.Scan(&rec.ActorEmail, &rec.Action, &rec.At, &raw); err != nil {
			return nil, fmt.Errorf("postgres: scan audit record: %w", err)
		}
		if err := json.Unmarshal(raw, &rec.Metadata); err != nil {
			return nil, fmt.Errorf("postgres: decode audit metadata: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// escapeLike makes the text literal inside a LIKE pattern.
func escapeLike(text string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(text)
}
