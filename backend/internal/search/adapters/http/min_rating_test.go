package searchhttp_test

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func rateBranch(t *testing.T, pool *pgxpool.Pool, branch int64, avg string, count int) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), `UPDATE branches SET rating_avg = $2::numeric, rating_count = $3 WHERE id = $1`, branch, avg, count); err != nil {
		t.Fatalf("rate branch: %v", err)
	}
}

func namesOf(t *testing.T, body []byte) []string {
	t.Helper()
	var decoded struct {
		Results []struct {
			Name string `json:"business_name"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	names := make([]string, 0, len(decoded.Results))
	for _, r := range decoded.Results {
		names = append(names, r.Name)
	}
	return names
}

func TestSearchFiltersByMinimumRatingWithoutReorderingTheResults(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	rateBranch(t, pool, addBranch(t, pool, 1, 100), "4.50", 6)
	rateBranch(t, pool, addBranch(t, pool, 2, 200), "3.00", 6)
	rateBranch(t, pool, addBranch(t, pool, 3, 300), "5.00", 1) // reads "Nuevo": no stars to claim
	rateBranch(t, pool, addBranch(t, pool, 4, 400), "4.20", 9)

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"no filter", origin, []string{"Studio 1", "Studio 2", "Studio 3", "Studio 4"}},
		{"4 stars, still by distance", origin + "&min_rating=4", []string{"Studio 1", "Studio 4"}},
		{"half a star", origin + "&min_rating=4.5", []string{"Studio 1"}},
		{"0 is no filter", origin + "&min_rating=0", []string{"Studio 1", "Studio 2", "Studio 3", "Studio 4"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := get(h, tt.query)
			if rec.Code != 200 {
				t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
			}
			got := namesOf(t, rec.Body.Bytes())
			if len(got) != len(tt.want) {
				t.Fatalf("results = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("results = %v, want %v", got, tt.want)
					break
				}
			}
		})
	}
}

func TestSearchRefusesAMinimumRatingItCannotUse(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	addBranch(t, pool, 1, 100)

	for _, bad := range []string{"abc", "6", "5.1", "-1", "NaN", "Inf"} {
		rec := get(h, origin+"&min_rating="+url.QueryEscape(bad))
		var body struct {
			Error struct {
				Code    string         `json:"code"`
				Details map[string]any `json:"details"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: decode: %v", bad, err)
		}
		if rec.Code != 422 || body.Error.Code != "validation_failed" || body.Error.Details["field"] != "min_rating" {
			t.Errorf("min_rating=%s: %d %s, want 422 validation_failed on min_rating", bad, rec.Code, rec.Body)
		}
	}
}
