package searchhttp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
	searchhttp "github.com/joaquinBrassinne/MOVO/internal/search/adapters/http"
	searchpostgres "github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

const origin = "lat=-32.17&lng=-64.11"

func newServer(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()

	pool := dbtest.New(t)
	branches := searchpostgres.NewBranches(pool)
	record := commands.NewRecordMetric(searchpostgres.NewMetrics(pool), nil)
	h := searchhttp.New(queries.NewNearbyBranches(branches), queries.NewFeaturedBranches(branches, nil), nil).
		WithWhatsApp(commands.NewOpenWhatsApp(branches, record, nil)).
		WithProfile(queries.NewBusinessProfile(branches, storage.NewMemory()))
	return httpx.NewRouter(nil, nil, h.Mount), pool
}

// addBranch inserts an active business with one branch distM meters north of the origin.
func addBranch(t *testing.T, pool *pgxpool.Pool, n int, distM float64) (branch int64) {
	t.Helper()

	var owner, business int64
	if err := pool.QueryRow(t.Context(),
		`INSERT INTO users (email, password_hash, role) VALUES ($1, 'hash', 'business_admin') RETURNING id`,
		fmt.Sprintf("owner%d@example.com", n)).Scan(&owner); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	if err := pool.QueryRow(t.Context(),
		`INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status, verified_at)
		 VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), $2, $3, '5491112345678', 'active', now()) RETURNING id`,
		owner, fmt.Sprintf("Studio %d", n), fmt.Sprintf("studio-%d", n)).Scan(&business); err != nil {
		t.Fatalf("insert business: %v", err)
	}
	if err := pool.QueryRow(t.Context(),
		`INSERT INTO branches (business_id, name, address, city, location)
		 VALUES ($1, 'Centro', 'Calle 1', 'Río Tercero', ST_SetSRID(ST_MakePoint(-64.11, $2::float8), 4326)::geography) RETURNING id`,
		business, -32.17+distM/111_000.0).Scan(&branch); err != nil {
		t.Fatalf("insert branch: %v", err)
	}
	return branch
}

func get(h http.Handler, query string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/search?"+query, nil))
	return rec
}

func TestSearchReturnsALightCardPerBranch(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	addBranch(t, pool, 1, 1_000)

	rec := get(h, origin)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var body struct {
		Results    []map[string]any `json:"results"`
		NextOffset *int             `json:"next_offset"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Results) != 1 || body.NextOffset != nil {
		t.Fatalf("results = %v, next = %v; want one result and no next page", body.Results, body.NextOffset)
	}
	card := body.Results[0]
	if card["business_name"] != "Studio 1" || card["business_slug"] != "studio-1" || card["verified"] != true || card["category_slug"] != "yoga" || card["category_name"] != "Yoga" {
		t.Errorf("card = %v", card)
	}
	if d, _ := card["distance_m"].(float64); d < 990 || d > 1_010 {
		t.Errorf("distance_m = %v, want about 1000", card["distance_m"])
	}
	for _, internal := range []string{"whatsapp_number", "whatsapp", "business_id", "owner_user_id", "status"} {
		if _, found := card[internal]; found {
			t.Errorf("card exposes %q", internal)
		}
	}
	if strings.Contains(rec.Body.String(), "5491112345678") {
		t.Error("the response contains the WhatsApp number")
	}
}

func TestSearchPagesWithNextOffset(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	for i := 1; i <= 3; i++ {
		addBranch(t, pool, i, float64(i)*100)
	}

	var first struct {
		Results    []any `json:"results"`
		NextOffset *int  `json:"next_offset"`
	}
	rec := get(h, origin+"&limit=2")
	if err := json.Unmarshal(rec.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(first.Results) != 2 || first.NextOffset == nil || *first.NextOffset != 2 {
		t.Fatalf("first page = %s", rec.Body)
	}

	var second struct {
		Results    []any `json:"results"`
		NextOffset *int  `json:"next_offset"`
	}
	rec = get(h, fmt.Sprintf("%s&limit=2&offset=%d", origin, *first.NextOffset))
	if err := json.Unmarshal(rec.Body.Bytes(), &second); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(second.Results) != 1 || second.NextOffset != nil {
		t.Errorf("second page = %s", rec.Body)
	}
}

func TestSearchAnswersAnEmptyListNotNull(t *testing.T) {
	t.Parallel()

	h, _ := newServer(t)
	rec := get(h, origin)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"results":[]`) {
		t.Errorf("status = %d, body %s; want 200 and an empty list", rec.Code, rec.Body)
	}
}

func TestSearchRejectsBadParameters(t *testing.T) {
	t.Parallel()

	h, _ := newServer(t)
	for name, tc := range map[string]struct{ query, field, reason string }{
		"missing lat":     {"lng=-64.11", "lat", "required"},
		"missing lng":     {"lat=-32.17", "lng", "required"},
		"text latitude":   {"lat=abc&lng=-64.11", "lat", "invalid"},
		"latitude range":  {"lat=91&lng=-64.11", "lat", "out_of_range"},
		"longitude range": {"lat=-32.17&lng=181", "lng", "out_of_range"},
		"radius range":    {origin + "&radius_m=999999", "radius_m", "out_of_range"},
		"text radius":     {origin + "&radius_m=big", "radius_m", "invalid"},
		"limit range":     {origin + "&limit=500", "limit", "out_of_range"},
		"offset range":    {origin + "&offset=-1", "offset", "out_of_range"},
	} {
		rec := get(h, tc.query)
		var body httpx.ErrorEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		details, _ := body.Error.Details.(map[string]any)
		if rec.Code != http.StatusUnprocessableEntity || body.Error.Code != searchhttp.CodeValidationFailed ||
			details["field"] != tc.field || details["reason"] != tc.reason {
			t.Errorf("%s: status %d, body %s; want 422 on %s/%s", name, rec.Code, rec.Body, tc.field, tc.reason)
		}
	}
}

func getFeatured(h http.Handler, query string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/home/featured?"+query, nil))
	return rec
}

func TestFeaturedAnswersAnEmptyListUntilBusinessesAreFeatured(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	addBranch(t, pool, 1, 500)

	rec := getFeatured(h, origin)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"results":[]`) {
		t.Errorf("status = %d, body %s; want 200 and an empty list", rec.Code, rec.Body)
	}
}

func TestFeaturedRejectsBadParameters(t *testing.T) {
	t.Parallel()

	h, _ := newServer(t)
	for name, tc := range map[string]struct{ query, field, reason string }{
		"missing lat": {"lng=-64.11", "lat", "required"},
		"limit range": {origin + "&limit=21", "limit", "out_of_range"},
		"text radius": {origin + "&radius_m=big", "radius_m", "invalid"},
	} {
		rec := getFeatured(h, tc.query)
		var body httpx.ErrorEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		details, _ := body.Error.Details.(map[string]any)
		if rec.Code != http.StatusUnprocessableEntity || details["field"] != tc.field || details["reason"] != tc.reason {
			t.Errorf("%s: status %d, body %s; want 422 on %s/%s", name, rec.Code, rec.Body, tc.field, tc.reason)
		}
	}
}
