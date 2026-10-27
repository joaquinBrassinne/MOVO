// Package advisory runs a job under a PostgreSQL advisory lock, so a scheduled task
// does not run twice when more than one instance of its binary is up (SDD 3).
package advisory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunExclusive runs job only if no other session holds the lock for key. It returns
// ran=false, without error, when somebody else does. The lock lives on one dedicated
// connection and is released when the job ends, and also if the process dies, because
// PostgreSQL drops a session's locks with its connection.
func RunExclusive(ctx context.Context, pool *pgxpool.Pool, key int64, job func(context.Context) error) (ran bool, err error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("advisory: acquire connection: %w", err)
	}
	defer conn.Release()

	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key).Scan(&locked); err != nil {
		return false, fmt.Errorf("advisory: try lock: %w", err)
	}
	if !locked {
		return false, nil
	}
	defer func() {
		// The caller's context may be cancelled already; unlock must still go out.
		if _, uerr := conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, key); uerr != nil {
			// A connection that cannot unlock must not return to the pool holding the lock.
			_ = conn.Conn().Close(context.WithoutCancel(ctx))
		}
	}()

	return true, job(ctx)
}
