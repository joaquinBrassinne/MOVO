package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
)

// Alerts implements billing.OwnerAlerts. The owner is always taken from the token, and
// both queries reach the business through businesses.owner_user_id, so there is no way to
// name another business.
type Alerts struct {
	pool *pgxpool.Pool
}

// NewAlerts builds the reader over pool.
func NewAlerts(pool *pgxpool.Pool) *Alerts { return &Alerts{pool: pool} }

// Unread returns the unread alerts of the owner's business, newest first.
func (r *Alerts) Unread(ctx context.Context, ownerID int64) ([]billing.Alert, error) {
	rows, err := r.pool.Query(ctx, `
SELECT a.id, a.type, a.payload, a.created_at
FROM alerts a
JOIN businesses b ON b.id = a.business_id
WHERE b.owner_user_id = $1 AND a.read_at IS NULL
ORDER BY a.created_at DESC, a.id DESC`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list alerts: %w", err)
	}
	defer rows.Close()

	out := []billing.Alert{}
	for rows.Next() {
		var (
			alert billing.Alert
			raw   []byte
		)
		if err := rows.Scan(&alert.ID, &alert.Type, &raw, &alert.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan alert: %w", err)
		}
		if err := json.Unmarshal(raw, &alert.Payload); err != nil {
			return nil, fmt.Errorf("postgres: decode alert payload: %w", err)
		}
		out = append(out, alert)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list alerts: %w", err)
	}
	return out, nil
}

// MarkRead dismisses an alert of the owner's business.
func (r *Alerts) MarkRead(ctx context.Context, ownerID, alertID int64) error {
	var found int64
	err := r.pool.QueryRow(ctx, `
WITH mine AS (
    SELECT a.id FROM alerts a JOIN businesses b ON b.id = a.business_id
    WHERE a.id = $2 AND b.owner_user_id = $1
), upd AS (
    UPDATE alerts SET read_at = now() WHERE id IN (SELECT id FROM mine) AND read_at IS NULL
)
SELECT id FROM mine`, ownerID, alertID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return billing.ErrAlertNotFound
	}
	if err != nil {
		return fmt.Errorf("postgres: mark alert read: %w", err)
	}
	return nil
}
