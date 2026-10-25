package searchhttp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getProfile(h http.Handler, slug string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/businesses/"+slug, nil))
	return rec
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

type slotBody struct {
	Weekday int    `json:"weekday"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

type profileBody struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Verified bool   `json:"verified"`
	Category struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
	} `json:"category"`
	Branches []struct {
		BranchID int64      `json:"branch_id"`
		Hours    []slotBody `json:"hours"`
	} `json:"branches"`
	Services []struct {
		ID       int64      `json:"id"`
		Name     string     `json:"name"`
		BranchID *int64     `json:"branch_id"`
		Price    *int       `json:"price"`
		Schedule []slotBody `json:"schedule"`
	} `json:"services"`
	Images []string `json:"images"`
}

func TestProfileShowsOnlyWhatIsPublic(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	branch := addBranch(t, pool, 1, 1_000)
	var business int64
	if err := pool.QueryRow(t.Context(), `SELECT business_id FROM branches WHERE id = $1`, branch).Scan(&business); err != nil {
		t.Fatalf("business of branch: %v", err)
	}

	// A branch that was retired, and a service tied to it, must not show.
	var retired int64
	if err := pool.QueryRow(t.Context(),
		`INSERT INTO branches (business_id, name, address, city, location, active)
		 VALUES ($1, 'Cerrada', 'Calle 9', 'Río Tercero', ST_SetSRID(ST_MakePoint(-64.11, -32.17), 4326)::geography, false) RETURNING id`,
		business).Scan(&retired); err != nil {
		t.Fatalf("insert retired branch: %v", err)
	}
	exec(t, pool, `INSERT INTO business_hours (branch_id, weekday, opens_at, closes_at) VALUES ($1, 1, '16:00', '20:00'), ($1, 1, '09:00', '13:00')`, branch)
	exec(t, pool, `INSERT INTO business_hours (branch_id, weekday, opens_at, closes_at) VALUES ($1, 2, '09:00', '13:00')`, retired)
	exec(t, pool, `INSERT INTO services (business_id, name, price) VALUES ($1, 'Yoga', 8000)`, business)
	exec(t, pool, `INSERT INTO services (business_id, name, active) VALUES ($1, 'Pausado', false)`, business)
	exec(t, pool, `INSERT INTO services (business_id, branch_id, name) VALUES ($1, $2, 'De la sede cerrada')`, business, retired)
	exec(t, pool, `INSERT INTO service_schedules (service_id, weekday, starts_at, ends_at)
	               SELECT id, 3, '10:00', '11:00' FROM services WHERE name = 'Yoga'`)
	exec(t, pool, `INSERT INTO business_images (business_id, object_key, position) VALUES ($1, 'b/second.jpg', 1), ($1, 'b/first.jpg', 0)`, business)

	rec := getProfile(h, "studio-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var body profileBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.Name != "Studio 1" || body.Slug != "studio-1" || !body.Verified || body.Category.Slug != "yoga" || body.Category.Name == "" {
		t.Errorf("header = %+v", body)
	}
	if len(body.Branches) != 1 || body.Branches[0].BranchID != branch {
		t.Fatalf("branches = %+v, want only the public one", body.Branches)
	}
	if hours := body.Branches[0].Hours; len(hours) != 2 || hours[0].Start != "09:00" || hours[1].Start != "16:00" || hours[0].Weekday != 1 {
		t.Errorf("hours = %+v, want the two Monday slots in time order", hours)
	}
	if len(body.Services) != 1 || body.Services[0].Name != "Yoga" {
		t.Fatalf("services = %+v, want only the active one of a public branch", body.Services)
	}
	yoga := body.Services[0]
	if yoga.BranchID != nil || yoga.Price == nil || *yoga.Price != 8000 || len(yoga.Schedule) != 1 || yoga.Schedule[0].Start != "10:00" {
		t.Errorf("service = %+v", yoga)
	}
	if len(body.Images) != 2 || !strings.Contains(body.Images[0], "first.jpg") || !strings.Contains(body.Images[1], "second.jpg") {
		t.Errorf("images = %v, want first then second", body.Images)
	}
	for _, leak := range []string{"5491112345678", "whatsapp", "owner", "status", "Cerrada", "Pausado"} {
		if strings.Contains(rec.Body.String(), leak) {
			t.Errorf("the response contains %q", leak)
		}
	}
}

func TestProfileOfABusinessWithoutServicesOrPhotosHasEmptyLists(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	addBranch(t, pool, 1, 1_000)

	rec := getProfile(h, "studio-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, field := range []string{"services", "images"} {
		if string(raw[field]) != "[]" {
			t.Errorf("%s = %s, want []", field, raw[field])
		}
	}
}

func TestProfileOfABusinessThatIsNotPublicIsNotFound(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	branch := addBranch(t, pool, 1, 1_000)
	addBranch(t, pool, 2, 1_000)

	tests := []struct {
		name string
		prep func()
		slug string
	}{
		{"unknown slug", func() {}, "nobody"},
		{"business under review", func() {
			exec(t, pool, `UPDATE businesses SET status = 'pending_review' WHERE slug = 'studio-2'`)
		}, "studio-2"},
		{"business paused", func() {
			exec(t, pool, `UPDATE businesses SET status = 'paused' WHERE slug = 'studio-2'`)
		}, "studio-2"},
		{"every branch retired", func() {
			exec(t, pool, `UPDATE branches SET active = false WHERE id = $1`, branch)
		}, "studio-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prep()
			rec := getProfile(h, tt.slug)
			if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"business_not_found"`) {
				t.Errorf("status = %d, body %s; want 404 business_not_found", rec.Code, rec.Body)
			}
		})
	}
}
