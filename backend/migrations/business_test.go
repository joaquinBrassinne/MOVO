package migrations_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

const (
	foreignKeyViolation = "23503"
	exclusionViolation  = "23P01"
)

func insertOwner(ctx context.Context, t *testing.T, pool *pgxpool.Pool, email string) int64 {
	t.Helper()

	var id int64
	err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1, 'hash', 'business_admin') RETURNING id`,
		email).Scan(&id)
	if err != nil {
		t.Fatalf("insert owner %s: %v", email, err)
	}
	return id
}

func insertBusiness(ctx context.Context, pool *pgxpool.Pool, owner int64, slug, whatsapp, status string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx,
		`INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status)
		 VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), 'Studio', $2, $3, $4) RETURNING id`,
		owner, slug, whatsapp, status).Scan(&id)
	return id, err
}

func requireSQLState(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("statement succeeded, want SQLSTATE %s", want)
	}
	if got := sqlState(t, err); got != want {
		t.Errorf("SQLSTATE = %s, want %s", got, want)
	}
}

// TestCategoriesAreSeeded pins the catalog the onboarding form offers.
func TestCategoriesAreSeeded(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)

	var got []string
	rows, err := pool.Query(t.Context(), `SELECT slug FROM categories ORDER BY slug`)
	if err != nil {
		t.Fatalf("query categories: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, slug)
	}

	want := []string{"entrenadores", "estetica", "gimnasios", "natacion", "padel", "pilates", "trekking-outdoor", "yoga"}
	if len(got) != len(want) {
		t.Fatalf("categories = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("categories = %v, want %v", got, want)
		}
	}
}

// TestBusinessesEnforceTheirRules pins one business per owner, a unique slug,
// digits-only WhatsApp numbers with country code and the known statuses.
func TestBusinessesEnforceTheirRules(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	owner := insertOwner(ctx, t, pool, "owner@example.com")
	other := insertOwner(ctx, t, pool, "other@example.com")

	if _, err := insertBusiness(ctx, pool, owner, "studio", "5491112345678", "draft"); err != nil {
		t.Fatalf("insert a valid business: %v", err)
	}

	tests := []struct {
		name     string
		owner    int64
		slug     string
		whatsapp string
		status   string
		want     string
	}{
		{"second business for the same owner", owner, "second", "5491112345678", "draft", uniqueViolation},
		{"duplicate slug", other, "studio", "5491112345678", "draft", uniqueViolation},
		{"slug with uppercase", other, "Studio-2", "5491112345678", "draft", checkViolation},
		{"slug with spaces", other, "my studio", "5491112345678", "draft", checkViolation},
		{"whatsapp with plus sign", other, "plus", "+5491112345678", "draft", checkViolation},
		{"whatsapp too short", other, "short", "12345", "draft", checkViolation},
		{"whatsapp starting with zero", other, "zero", "0491112345678", "draft", checkViolation},
		{"unknown status", other, "status", "5491112345678", "approved", checkViolation},
		{"unknown owner", 999999, "ghost", "5491112345678", "draft", foreignKeyViolation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := insertBusiness(ctx, pool, tt.owner, tt.slug, tt.whatsapp, tt.status)
			requireSQLState(t, err, tt.want)
		})
	}
}

// TestBranchesStoreALocationAndAreIndexedForProximity pins that a branch needs a
// point and that the GIST index phase 3 searches through exists.
func TestBranchesStoreALocationAndAreIndexedForProximity(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "draft")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO branches (business_id, name, address, city, location)
		 VALUES ($1, 'Centro', 'Calle 1', 'Rosario', ST_SetSRID(ST_MakePoint(-60.65, -32.95), 4326)::geography)`,
		business)
	if err != nil {
		t.Fatalf("insert a branch with a location: %v", err)
	}

	var rating float64
	var count int
	var active bool
	if err := pool.QueryRow(ctx, `SELECT rating_avg, rating_count, active FROM branches`).Scan(&rating, &count, &active); err != nil {
		t.Fatalf("read defaults: %v", err)
	}
	if rating != 0 || count != 0 || !active {
		t.Errorf("defaults = (%v, %d, %v), want (0, 0, true)", rating, count, active)
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO branches (business_id, name, address, city) VALUES ($1, 'Sin pin', 'Calle 2', 'Rosario')`, business)
	requireSQLState(t, err, "23502")

	var indexed bool
	err = pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'branches' AND indexdef ILIKE '%USING gist%(location)%')`).Scan(&indexed)
	if err != nil {
		t.Fatalf("query indexes: %v", err)
	}
	if !indexed {
		t.Error("branches has no GIST index on location")
	}
}

// TestBranchesAndImagesFollowTheirBusiness pins the cascade and the ordering rules
// of hours and images.
func TestBranchesAndImagesFollowTheirBusiness(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "draft")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	var branch int64
	err = pool.QueryRow(ctx,
		`INSERT INTO branches (business_id, name, address, city, location)
		 VALUES ($1, 'Centro', 'Calle 1', 'Rosario', ST_SetSRID(ST_MakePoint(-60.65, -32.95), 4326)::geography) RETURNING id`,
		business).Scan(&branch)
	if err != nil {
		t.Fatalf("insert branch: %v", err)
	}

	hours := func(weekday int, opens, closes string) error {
		_, err := pool.Exec(ctx,
			`INSERT INTO business_hours (branch_id, weekday, opens_at, closes_at) VALUES ($1, $2, $3, $4)`,
			branch, weekday, opens, closes)
		return err
	}
	if err := hours(1, "09:00", "13:00"); err != nil {
		t.Fatalf("insert hours: %v", err)
	}
	if err := hours(1, "16:00", "20:00"); err != nil {
		t.Fatalf("a second slot on the same day must be allowed: %v", err)
	}
	requireSQLState(t, hours(7, "09:00", "13:00"), checkViolation)
	requireSQLState(t, hours(2, "13:00", "09:00"), checkViolation)
	requireSQLState(t, hours(1, "09:00", "10:00"), exclusionViolation)
	requireSQLState(t, hours(1, "12:00", "17:00"), exclusionViolation)
	if err := hours(1, "13:00", "16:00"); err != nil {
		t.Fatalf("a slot that starts where another ends must be allowed: %v", err)
	}

	image := func(key string, position int) error {
		_, err := pool.Exec(ctx,
			`INSERT INTO business_images (business_id, object_key, position) VALUES ($1, $2, $3)`,
			business, key, position)
		return err
	}
	if err := image("businesses/1/a.jpg", 0); err != nil {
		t.Fatalf("insert image: %v", err)
	}
	requireSQLState(t, image("businesses/1/a.jpg", 1), uniqueViolation)
	requireSQLState(t, image("businesses/1/b.jpg", 0), uniqueViolation)

	if _, err := pool.Exec(ctx, `DELETE FROM businesses WHERE id = $1`, business); err != nil {
		t.Fatalf("delete business: %v", err)
	}
	for _, table := range []string{"branches", "business_hours", "business_images"} {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != 0 {
			t.Errorf("%s has %d rows after deleting the business, want 0", table, n)
		}
	}
}
