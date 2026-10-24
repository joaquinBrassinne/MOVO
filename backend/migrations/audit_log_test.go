package migrations_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

const raiseException = "P0001"

// TestAuditLogIsAppendOnly pins that an entry can be written and read but never
// changed or removed, so the trail of super admin actions can be trusted.
func TestAuditLogIsAppendOnly(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	admin := insertOwner(ctx, t, pool, "admin@example.com")

	var id int64
	err := pool.QueryRow(ctx,
		`INSERT INTO audit_log (actor_id, action, entity, entity_id, metadata)
		 VALUES ($1, 'business.approved', 'business', 42, '{"from":"pending_review","to":"active"}') RETURNING id`,
		admin).Scan(&id)
	if err != nil {
		t.Fatalf("insert an entry: %v", err)
	}

	var createdSet bool
	var action string
	if err := pool.QueryRow(ctx, `SELECT action, created_at IS NOT NULL FROM audit_log WHERE id = $1`, id).Scan(&action, &createdSet); err != nil || action != "business.approved" || !createdSet {
		t.Fatalf("read back = %q, %v, %v", action, createdSet, err)
	}

	_, err = pool.Exec(ctx, `UPDATE audit_log SET action = 'tampered' WHERE id = $1`, id)
	requireSQLState(t, err, raiseException)
	_, err = pool.Exec(ctx, `DELETE FROM audit_log WHERE id = $1`, id)
	requireSQLState(t, err, raiseException)

	tests := []struct {
		name     string
		actor    int64
		action   string
		entity   string
		metadata string
		want     string
	}{
		{"blank action", admin, " ", "business", "{}", checkViolation},
		{"blank entity", admin, "x", "", "{}", checkViolation},
		{"metadata that is not an object", admin, "x", "business", "[1]", checkViolation},
		{"unknown actor", 999999, "x", "business", "{}", foreignKeyViolation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx,
				`INSERT INTO audit_log (actor_id, action, entity, entity_id, metadata) VALUES ($1, $2, $3, 1, $4::jsonb)`,
				tt.actor, tt.action, tt.entity, tt.metadata)
			requireSQLState(t, err, tt.want)
		})
	}
}
