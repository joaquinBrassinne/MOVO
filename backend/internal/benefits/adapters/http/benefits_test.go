package benefitshttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	benefitshttp "github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/ratelimit"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// tokens maps a bearer token to its caller, standing in for the identity module.
type tokens map[string]httpx.Principal

func (t tokens) Verify(raw string) (httpx.Principal, error) {
	p, ok := t[raw]
	if !ok {
		return httpx.Principal{}, errors.New("unknown token")
	}
	return p, nil
}

// owners resolves the business of an owner straight from the database, standing in for
// the adapter the composition root builds from the business module.
type owners struct{ pool *pgxpool.Pool }

func (o owners) EditableBusinessOf(ctx context.Context, ownerID int64) (int64, error) {
	var id int64
	var status string
	err := o.pool.QueryRow(ctx, `SELECT id, status FROM businesses WHERE owner_user_id = $1`, ownerID).Scan(&id, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, benefits.ErrBusinessNotFound
	}
	if err != nil {
		return 0, err
	}
	if status == "deleted" {
		return 0, benefits.ErrBusinessNotEditable
	}
	return id, nil
}

type server struct {
	t      *testing.T
	url    string
	pool   *pgxpool.Pool
	tokens tokens
}

func newServer(t *testing.T) *server {
	t.Helper()
	return newServerWithLimit(t, 1000)
}

// newServerWithLimit builds the stack with a coupon request budget per origin. Every
// request of a test comes from the same address, so the budget is the test's own.
func newServerWithLimit(t *testing.T, maxAttempts int) *server {
	t.Helper()

	pool := dbtest.New(t)
	s := &server{t: t, pool: pool, tokens: tokens{}}
	repo := postgres.NewBenefits(pool)
	res := owners{pool: pool}
	limiter := ratelimit.New(ratelimit.Config{MaxAttempts: maxAttempts, Window: time.Hour, MaxKeys: 100}, nil)
	handlers := benefitshttp.New(benefitshttp.Deps{
		Verifier:  s.tokens,
		List:      queries.NewListMyBenefits(res, repo),
		Add:       commands.NewAddBenefit(res, repo),
		Edit:      commands.NewEditBenefit(res, repo),
		SetActive: commands.NewSetBenefitActive(res, repo),
		Generate:  commands.NewGenerateCode(postgres.NewCoupons(pool), limiter, nil, nil),
		Redeem:    commands.NewRedeemCode(res, postgres.NewCoupons(pool), nil),
		Preview:   queries.NewPreviewCode(res, postgres.NewCoupons(pool), nil),
		View:      queries.NewViewMyCoupon(postgres.NewCoupons(pool), nil),
		Activity:  queries.NewRecentActivity(res, postgres.NewCoupons(pool)),
	})

	// The business module serves /me/business as a mounted router before this one is
	// mounted; the stub answers 418 to anything it receives, so a benefit route that
	// the mount swallowed would show up as a 418.
	business := func(r chi.Router) {
		r.Route("/me/business", func(me chi.Router) {
			me.HandleFunc("/*", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
		})
	}
	ts := httptest.NewServer(httpx.NewRouter(nil, nil, business, handlers.Mount))
	t.Cleanup(ts.Close)
	s.url = ts.URL + "/api/v1"
	return s
}

// user creates a user of the given role and returns its id and bearer token.
func (s *server) user(email, role string) (int64, string) {
	s.t.Helper()

	var id int64
	err := s.pool.QueryRow(s.t.Context(),
		`INSERT INTO users (email, password_hash, role) VALUES ($1, 'hash', $2) RETURNING id`, email, role).Scan(&id)
	if err != nil {
		s.t.Fatalf("create user %s: %v", email, err)
	}
	token := "token-" + email
	s.tokens[token] = httpx.Principal{UserID: id, Role: role}
	return id, token
}

// owner creates an owner with a business in the given status and returns its token.
func (s *server) owner(email, status string) string {
	s.t.Helper()

	id, token := s.user(email, "business_admin")
	_, err := s.pool.Exec(s.t.Context(), `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), 'Studio', $2, '5491112345678', $3)`,
		id, "studio-"+strconv.FormatInt(id, 10), status)
	if err != nil {
		s.t.Fatalf("create business: %v", err)
	}
	return token
}

type reply struct {
	status int
	header http.Header
	body   map[string]any
}

func (s *server) do(method, path, token string, body any) reply {
	s.t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			s.t.Fatalf("encode body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(s.t.Context(), method, s.url+path, reader)
	if err != nil {
		s.t.Fatalf("build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		s.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()

	out := reply{status: res.StatusCode, header: res.Header}
	raw, _ := io.ReadAll(res.Body)
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out.body); err != nil {
			s.t.Fatalf("%s %s: response is not a JSON object: %s", method, path, raw)
		}
	}
	return out
}

func (r reply) errorCode() string {
	e, _ := r.body["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

func (r reply) errorField() string {
	e, _ := r.body["error"].(map[string]any)
	d, _ := e["details"].(map[string]any)
	field, _ := d["field"].(string)
	return field
}

func benefitPath(r reply) string {
	return "/me/business/benefits/" + strconv.FormatInt(int64(r.body["id"].(float64)), 10)
}

func TestOwnerManagesBenefitsAndTheMonthlyLimit(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	owner := s.owner("owner@example.com", "active")

	percent := s.do(http.MethodPost, "/me/business/benefits", owner, map[string]any{"type": "percent", "percent": 15, "monthly_limit": 40})
	if percent.status != http.StatusCreated || percent.body["type"] != "percent" || percent.body["percent"] != float64(15) ||
		percent.body["gift_description"] != nil || percent.body["active"] != true || percent.body["monthly_limit"] != float64(40) {
		t.Fatalf("create percent = %d %v", percent.status, percent.body)
	}
	gift := s.do(http.MethodPost, "/me/business/benefits", owner, map[string]any{"type": "welcome_gift", "gift_description": " Una clase gratis ", "monthly_limit": 10})
	if gift.status != http.StatusCreated || gift.body["percent"] != nil || gift.body["gift_description"] != "Una clase gratis" {
		t.Fatalf("create gift = %d %v", gift.status, gift.body)
	}

	// Turning the percent benefit into a gift drops the percent, and the limit changes.
	edited := s.do(http.MethodPut, benefitPath(percent), owner, map[string]any{"type": "welcome_gift", "gift_description": "Mate", "monthly_limit": 25})
	if edited.status != http.StatusOK || edited.body["percent"] != nil || edited.body["gift_description"] != "Mate" || edited.body["monthly_limit"] != float64(25) {
		t.Fatalf("edit = %d %v", edited.status, edited.body)
	}
	if off := s.do(http.MethodPut, benefitPath(percent)+"/active", owner, map[string]any{"active": false}); off.status != http.StatusOK || off.body["active"] != false {
		t.Fatalf("deactivate = %d %v", off.status, off.body)
	}
	if again := s.do(http.MethodPut, benefitPath(percent)+"/active", owner, map[string]any{"active": false}); again.status != http.StatusOK || again.body["active"] != false {
		t.Fatalf("deactivate twice = %d %v, want it idempotent", again.status, again.body)
	}
	// Editing keeps the active flag.
	if kept := s.do(http.MethodPut, benefitPath(percent), owner, map[string]any{"type": "percent", "percent": 20, "monthly_limit": 5}); kept.body["active"] != false || kept.body["gift_description"] != nil {
		t.Errorf("edit = %v, want it to stay inactive with no gift", kept.body)
	}

	list := s.do(http.MethodGet, "/me/business/benefits", owner, nil)
	items, _ := list.body["benefits"].([]any)
	if list.status != http.StatusOK || len(items) != 2 {
		t.Fatalf("list = %d %v, want both benefits, the inactive one included", list.status, list.body)
	}
	if first, _ := items[0].(map[string]any); first["percent"] != float64(20) || first["active"] != false {
		t.Errorf("first benefit = %v, want the oldest first", first)
	}
	if list.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", list.header.Get("Cache-Control"))
	}
}

func TestEmptyListIsAnArrayNotNull(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	owner := s.owner("owner@example.com", "draft")

	list := s.do(http.MethodGet, "/me/business/benefits", owner, nil)
	if items, ok := list.body["benefits"].([]any); !ok || len(items) != 0 {
		t.Fatalf("list = %d %v, want an empty array", list.status, list.body)
	}
}

func TestBenefitValidationNamesTheField(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	owner := s.owner("owner@example.com", "active")

	for _, tc := range []struct {
		name  string
		body  map[string]any
		field string
	}{
		{"percent 25", map[string]any{"type": "percent", "percent": 25, "monthly_limit": 5}, "percent"},
		{"percent missing", map[string]any{"type": "percent", "monthly_limit": 5}, "percent"},
		{"gift missing", map[string]any{"type": "welcome_gift", "monthly_limit": 5}, "gift_description"},
		{"unknown type", map[string]any{"type": "two_for_one", "monthly_limit": 5}, "type"},
		{"no type", map[string]any{"percent": 10, "monthly_limit": 5}, "type"},
		{"no limit", map[string]any{"type": "percent", "percent": 10}, "monthly_limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := s.do(http.MethodPost, "/me/business/benefits", owner, tc.body)
			if got.status != http.StatusUnprocessableEntity || got.errorCode() != "validation_failed" || got.errorField() != tc.field {
				t.Errorf("create = %d %q field %q, want 422 validation_failed on %q", got.status, got.errorCode(), got.errorField(), tc.field)
			}
		})
	}
	if list := s.do(http.MethodGet, "/me/business/benefits", owner, nil); len(list.body["benefits"].([]any)) != 0 {
		t.Error("a rejected benefit must not be stored")
	}
}

func TestOwnerCannotReachTheBenefitOfAnotherBusiness(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	mine := s.owner("mine@example.com", "active")
	other := s.owner("other@example.com", "active")
	theirs := s.do(http.MethodPost, "/me/business/benefits", other, map[string]any{"type": "percent", "percent": 10, "monthly_limit": 5})

	for _, call := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPut, benefitPath(theirs), map[string]any{"type": "percent", "percent": 20, "monthly_limit": 9}},
		{http.MethodPut, benefitPath(theirs) + "/active", map[string]any{"active": false}},
		{http.MethodPut, "/me/business/benefits/999999", map[string]any{"type": "percent", "percent": 10, "monthly_limit": 1}},
		{http.MethodPut, "/me/business/benefits/abc/active", map[string]any{"active": true}},
	} {
		got := s.do(call.method, call.path, mine, call.body)
		if got.status != http.StatusNotFound || got.errorCode() != "benefit_not_found" {
			t.Errorf("%s %s = %d %q, want 404 benefit_not_found", call.method, call.path, got.status, got.errorCode())
		}
	}

	// The business id never comes from the body: the strict decoder refuses a forged one.
	forged := s.do(http.MethodPost, "/me/business/benefits", mine, map[string]any{"type": "percent", "percent": 10, "monthly_limit": 5, "business_id": 1})
	if forged.status != http.StatusBadRequest {
		t.Fatalf("forged business_id = %d %v, want 400", forged.status, forged.body)
	}
	own := s.do(http.MethodPost, "/me/business/benefits", mine, map[string]any{"type": "percent", "percent": 10, "monthly_limit": 5})
	if own.status != http.StatusCreated {
		t.Fatalf("create = %d %v", own.status, own.body)
	}
	if list := s.do(http.MethodGet, "/me/business/benefits", other, nil); len(list.body["benefits"].([]any)) != 1 {
		t.Errorf("the other business lists %v, want only its own benefit", list.body)
	}
	if untouched := s.do(http.MethodGet, "/me/business/benefits", other, nil).body["benefits"].([]any)[0].(map[string]any); untouched["percent"] != float64(10) || untouched["active"] != true {
		t.Errorf("the other business's benefit changed: %v", untouched)
	}
}

func TestBenefitsNeedABusinessThatCanStillChange(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	_, noBusiness := s.user("nobody@example.com", "business_admin")
	deleted := s.owner("deleted@example.com", "deleted")
	body := map[string]any{"type": "percent", "percent": 10, "monthly_limit": 5}

	if got := s.do(http.MethodPost, "/me/business/benefits", noBusiness, body); got.status != http.StatusNotFound || got.errorCode() != "business_not_found" {
		t.Errorf("no business = %d %q, want 404 business_not_found", got.status, got.errorCode())
	}
	if got := s.do(http.MethodPost, "/me/business/benefits", deleted, body); got.status != http.StatusConflict || got.errorCode() != "business_deleted" {
		t.Errorf("deleted business = %d %q, want 409 business_deleted", got.status, got.errorCode())
	}
}

func TestBenefitRoutesNeedAnOwner(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	_, admin := s.user("admin@example.com", "super_admin")

	if got := s.do(http.MethodGet, "/me/business/benefits", "", nil); got.status != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", got.status)
	}
	if got := s.do(http.MethodGet, "/me/business/benefits", "token-unknown", nil); got.status != http.StatusUnauthorized {
		t.Errorf("bad token = %d, want 401", got.status)
	}
	if got := s.do(http.MethodPost, "/me/business/benefits", admin, map[string]any{"type": "percent", "percent": 10, "monthly_limit": 5}); got.status != http.StatusForbidden {
		t.Errorf("super admin = %d, want 403", got.status)
	}
}
