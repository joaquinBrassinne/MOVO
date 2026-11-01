package main

import (
	"fmt"
	"testing"
	"time"

	identitydomain "github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// metricsWorld is the setup shared by the end-to-end tests of the Pro metrics: businesses on
// each kind of plan, with coupons and counters seeded at chosen ages.
type metricsWorld struct {
	t     *testing.T
	h     *apiHarness
	admin string
	codes int
}

// metricsShop is one business with its owner's token and ids.
type metricsShop struct {
	token                            string
	owner, business, branch, benefit int64
}

const (
	planStarter = "starter"
	planPro     = "pro"
	planPilot   = "pilot"
)

// customerName and customerPhone are what no metric may ever show.
const (
	customerName  = "Ana García"
	customerPhone = "5491187654321"
)

func newMetricsWorld(t *testing.T) *metricsWorld {
	t.Helper()
	h := newAPIHarness(t)
	w := &metricsWorld{t: t, h: h, admin: h.user("root@example.com", identitydomain.RoleSuperAdmin)}
	if _, err := h.pool.Exec(t.Context(), `INSERT INTO customers (name, phone) VALUES ($1, $2)`, customerName, customerPhone); err != nil {
		t.Fatal(err)
	}
	return w
}

func (w *metricsWorld) must(err error) {
	w.t.Helper()
	if err != nil {
		w.t.Fatal(err)
	}
}

// shop creates an active business with one branch and one benefit, on the plan given: starter
// has no subscription, pro has a Pro one, pilot is flagged as a pilot with none.
func (w *metricsWorld) shop(email, slug, plan string) *metricsShop {
	w.t.Helper()
	ctx := w.t.Context()
	s := &metricsShop{token: w.h.user(email, identitydomain.RoleBusinessAdmin)}
	w.must(w.h.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&s.owner))
	w.must(w.h.pool.QueryRow(ctx, `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status, pilot)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), $2, $2, '5491112345678', 'active', $3) RETURNING id`,
		s.owner, slug, plan == planPilot).Scan(&s.business))
	w.must(w.h.pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Centro', 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography) RETURNING id`, s.business).Scan(&s.branch))
	w.must(w.h.pool.QueryRow(ctx, `INSERT INTO benefits (business_id, type, percent) VALUES ($1, 'percent', 10) RETURNING id`, s.business).Scan(&s.benefit))
	if plan == planPro {
		w.setPlan(s, planPro)
	}
	return s
}

// setPlan gives the business a subscription on the plan, or changes the one it has.
func (w *metricsWorld) setPlan(s *metricsShop, plan string) {
	w.t.Helper()
	var admin int64
	w.must(w.h.pool.QueryRow(w.t.Context(), `SELECT id FROM users WHERE email = 'root@example.com'`).Scan(&admin))
	_, err := w.h.pool.Exec(w.t.Context(), `
INSERT INTO subscriptions (business_id, plan, paid_until, confirmed_by) VALUES ($1, $2, CURRENT_DATE + 30, $3)
ON CONFLICT (business_id) DO UPDATE SET plan = EXCLUDED.plan`, s.business, plan, admin)
	w.must(err)
}

// coupons seeds n coupons created at the given age, redeemed (at the shop's branch) or expired.
func (w *metricsWorld) coupons(s *metricsShop, n int, redeemed bool, age time.Duration) {
	w.t.Helper()
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	createdAt := time.Now().Add(-age)
	for range n {
		w.codes++
		code := "YYYYYY" + string(letters[w.codes/len(letters)]) + string(letters[w.codes%len(letters)])
		hash := make([]byte, 32)
		copy(hash, fmt.Sprintf("%s-%d", code, s.business))
		var err error
		if redeemed {
			_, err = w.h.pool.Exec(w.t.Context(), `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ($1, $2, $3, (SELECT id FROM customers LIMIT 1), $4, 'redeemed', $5, $5, $5::timestamptz + interval '1 minute', $6, $7)`,
				code, s.benefit, s.business, hash, createdAt, s.owner, s.branch)
		} else {
			_, err = w.h.pool.Exec(w.t.Context(), `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at)
VALUES ($1, $2, $3, (SELECT id FROM customers LIMIT 1), $4, 'expired', $5, $5)`,
				code, s.benefit, s.business, hash, createdAt)
		}
		w.must(err)
	}
}

// counter adds n to a daily counter daysAgo days back on Argentina's calendar. A zero branch
// or service means none.
func (w *metricsWorld) counter(s *metricsShop, metric string, daysAgo int, branch, service int64, n int) {
	w.t.Helper()
	var branchArg, serviceArg any
	if branch != 0 {
		branchArg = branch
	}
	if service != 0 {
		serviceArg = service
	}
	_, err := w.h.pool.Exec(w.t.Context(), `
INSERT INTO metric_daily (business_id, branch_id, service_id, date, metric, count)
VALUES ($1, $2, $3, (now() AT TIME ZONE 'America/Argentina/Cordoba')::date - $4::int, $5, $6)`,
		s.business, branchArg, serviceArg, daysAgo, metric, n)
	w.must(err)
}

const day = 24 * time.Hour
