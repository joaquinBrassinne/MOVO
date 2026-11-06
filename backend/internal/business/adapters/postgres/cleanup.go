package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Cleanup implements business.ObjectCleanup. Keys are queued by Images.Remove, inside
// the transaction that deletes the photo row.
type Cleanup struct {
	pool *pgxpool.Pool
}

// NewCleanup builds the queue over pool.
func NewCleanup(pool *pgxpool.Pool) *Cleanup {
	return &Cleanup{pool: pool}
}

// Pending returns up to limit queued keys, never tried or tried longest ago first.
func (c *Cleanup) Pending(ctx context.Context, limit int) ([]string, error) {
	rows, err := c.pool.Query(ctx,
		`SELECT object_key FROM object_cleanup ORDER BY last_attempt_at NULLS FIRST, queued_at, object_key LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres: pending object deletes: %w", err)
	}
	defer rows.Close()

	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("postgres: scan object key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: read pending object deletes: %w", err)
	}
	return keys, nil
}

// Done drops the key from the queue. Dropping one that is not there is not an error.
func (c *Cleanup) Done(ctx context.Context, key string) error {
	if _, err := c.pool.Exec(ctx, `DELETE FROM object_cleanup WHERE object_key = $1`, key); err != nil {
		return fmt.Errorf("postgres: drop queued object %q: %w", key, err)
	}
	return nil
}

// Failed records one more failed attempt for the key.
func (c *Cleanup) Failed(ctx context.Context, key, reason string) error {
	if _, err := c.pool.Exec(ctx,
		`UPDATE object_cleanup SET attempts = attempts + 1, last_error = $2, last_attempt_at = now() WHERE object_key = $1`,
		key, reason); err != nil {
		return fmt.Errorf("postgres: record failed delete of %q: %w", key, err)
	}
	return nil
}
