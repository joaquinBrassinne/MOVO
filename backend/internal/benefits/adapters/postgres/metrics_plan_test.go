package postgres_test

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// seedManyCoupons fills benefit_codes with 40 businesses of 1000 coupons each, spread over
// 400 days, a third of them redeemed. Large enough for the planner to prefer an index to a
// full scan of the table.
func seedManyCoupons(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, sql := range []string{
		`INSERT INTO users (email, password_hash, role)
		 SELECT 'owner' || g || '@example.com', 'hash', 'business_admin' FROM generate_series(1, 40) g`,
		`INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status)
		 SELECT u.id, (SELECT id FROM categories WHERE slug = 'yoga'), 'B' || u.id, 'b' || u.id, '5491112345678', 'active'
		 FROM users u`,
		`INSERT INTO branches (business_id, name, address, city, location)
		 SELECT id, 'Centro', 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography FROM businesses`,
		`INSERT INTO benefits (business_id, type, percent) SELECT id, 'percent', 10 FROM businesses`,
		`INSERT INTO customers (name, phone) SELECT 'C' || g, (5491100000000 + g)::text FROM generate_series(1, 200) g`,
		`WITH rows AS (
		   SELECT b.id AS business_id, b.owner_user_id, br.id AS branch_id, bn.id AS benefit_id, g AS i,
		          row_number() OVER ()::bigint AS n
		   FROM businesses b
		   JOIN branches br ON br.business_id = b.id
		   JOIN benefits bn ON bn.business_id = b.id
		   CROSS JOIN generate_series(1, 1000) g
		 )
		 INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status,
		                            expires_at, created_at, redeemed_at, redeemed_by, redeemed_branch_id)
		 SELECT (SELECT string_agg(substr('ABCDEFGHJKLMNPQRSTUVWXYZ23456789', ((n / power(32, k)::bigint) % 32)::int + 1, 1), '' ORDER BY k)
		         FROM generate_series(0, 7) k),
		        benefit_id, business_id, (SELECT min(id) FROM customers) + (i % 200),
		        decode(md5(n || 'a') || md5(n || 'b'), 'hex'),
		        CASE WHEN i % 3 = 0 THEN 'redeemed' ELSE 'expired' END,
		        now() - (i % 400) * interval '1 day' + interval '48 hours',
		        now() - (i % 400) * interval '1 day',
		        CASE WHEN i % 3 = 0 THEN now() - (i % 400) * interval '1 day' + interval '2 hours' END,
		        CASE WHEN i % 3 = 0 THEN owner_user_id END,
		        CASE WHEN i % 3 = 0 THEN branch_id END
		 FROM rows`,
		`ANALYZE benefit_codes`,
	} {
		if _, err := pool.Exec(t.Context(), sql); err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
}

func explain(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) string {
	t.Helper()

	rows, err := pool.Query(t.Context(), "EXPLAIN "+sql, args...)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()

	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		plan.WriteString(line + "\n")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return plan.String()
}

// TestMetricsQueriesUseTheIndexesInsteadOfReadingEveryCoupon checks the access paths the
// business metrics rely on: a period by creation time, a period by redemption time, and the
// monthly aggregation of the last 12 months.
func TestMetricsQueriesUseTheIndexesInsteadOfReadingEveryCoupon(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	seedManyCoupons(t, pool)

	var business int64
	if err := pool.QueryRow(t.Context(), `SELECT min(id) FROM businesses`).Scan(&business); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, sql, index string
	}{
		{
			"coupons generated in 90 days",
			`SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND created_at >= now() - interval '90 days' AND created_at < now()`,
			"benefit_codes_business_created_idx",
		},
		{
			"coupons redeemed in 90 days",
			`SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= now() - interval '90 days' AND redeemed_at < now()`,
			"benefit_codes_business_redeemed_idx",
		},
		{
			"redemptions per month over the last 12 months",
			`SELECT date_trunc('month', redeemed_at AT TIME ZONE 'America/Argentina/Cordoba') AS month, count(*)
			 FROM benefit_codes WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= now() - interval '12 months'
			 GROUP BY 1`,
			"benefit_codes_business_redeemed_idx",
		},
	} {
		plan := explain(t, pool, tc.sql, business)
		if strings.Contains(plan, "Seq Scan on benefit_codes") {
			t.Errorf("%s reads the whole table:\n%s", tc.name, plan)
		}
		if !strings.Contains(plan, tc.index) {
			t.Errorf("%s does not use %s:\n%s", tc.name, tc.index, plan)
		}
	}
}
