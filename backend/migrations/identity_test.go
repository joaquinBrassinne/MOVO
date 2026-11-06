package migrations_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

const (
	uniqueViolation = "23505"
	checkViolation  = "23514"
)

func insertUser(ctx context.Context, t *testing.T, exec func(context.Context, string, ...any) error, email, role string) error {
	t.Helper()
	return exec(ctx, `INSERT INTO users (email, password_hash, role) VALUES ($1, 'hash', $2)`, email, role)
}

func sqlState(t *testing.T, err error) string {
	t.Helper()

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error = %v, want a PostgreSQL error", err)
	}
	return pgErr.Code
}

// TestUsersEnforceUniqueNormalizedEmailAndKnownRoles pins the rules the identity
// code relies on: one account per address, only normalized emails, two roles.
func TestUsersEnforceUniqueNormalizedEmailAndKnownRoles(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	exec := func(ctx context.Context, sql string, args ...any) error {
		_, err := pool.Exec(ctx, sql, args...)
		return err
	}

	if err := insertUser(ctx, t, exec, "owner@example.com", "business_admin"); err != nil {
		t.Fatalf("insert a valid user: %v", err)
	}

	tests := []struct {
		name  string
		email string
		role  string
		want  string
	}{
		{"duplicate email", "owner@example.com", "business_admin", uniqueViolation},
		{"uppercase email", "Owner2@example.com", "business_admin", checkViolation},
		{"padded email", " owner3@example.com", "business_admin", checkViolation},
		{"empty email", "", "business_admin", checkViolation},
		{"unknown role", "other@example.com", "customer", checkViolation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := insertUser(ctx, t, exec, tt.email, tt.role)
			if err == nil {
				t.Fatalf("insert (%q, %q) succeeded, want SQLSTATE %s", tt.email, tt.role, tt.want)
			}
			if got := sqlState(t, err); got != tt.want {
				t.Errorf("SQLSTATE = %s, want %s", got, tt.want)
			}
		})
	}
}

// TestRefreshTokensRequireHashedValueAndFollowTheirUser pins that only a 32-byte
// digest is storable, a digest is unique, and tokens vanish with their user.
func TestRefreshTokensRequireHashedValueAndFollowTheirUser(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()

	var userID int64
	err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ('a@example.com', 'hash', 'business_admin') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	digest := make([]byte, 32)
	insert := `INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`
	expires := time.Now().Add(time.Hour)

	if _, err := pool.Exec(ctx, insert, userID, digest, expires); err != nil {
		t.Fatalf("insert a valid refresh token: %v", err)
	}

	if _, err := pool.Exec(ctx, insert, userID, digest, expires); sqlState(t, err) != uniqueViolation {
		t.Errorf("duplicate digest: error = %v, want SQLSTATE %s", err, uniqueViolation)
	}
	if _, err := pool.Exec(ctx, insert, userID, []byte("short"), expires); sqlState(t, err) != checkViolation {
		t.Errorf("short digest: error = %v, want SQLSTATE %s", err, checkViolation)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	var remaining int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM refresh_tokens`).Scan(&remaining); err != nil {
		t.Fatalf("count refresh tokens: %v", err)
	}
	if remaining != 0 {
		t.Errorf("refresh tokens after deleting the user = %d, want 0", remaining)
	}
}
