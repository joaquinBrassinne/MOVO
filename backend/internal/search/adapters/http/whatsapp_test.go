package searchhttp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

const (
	browser = "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/126.0 Mobile Safari/537.36"
	number  = "5491112345678"
)

func clickRedirect(h http.Handler, path, userAgent string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/r/wa/"+path, nil)
	req.Header.Set("User-Agent", userAgent)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// clicks returns how many WhatsApp clicks are counted for a branch, summed over rows.
func clicks(t *testing.T, pool *pgxpool.Pool, branch int64) (total int64) {
	t.Helper()
	if err := pool.QueryRow(t.Context(),
		`SELECT COALESCE(sum(count), 0) FROM metric_daily WHERE branch_id = $1 AND metric = 'whatsapp_click'`, branch).Scan(&total); err != nil {
		t.Fatalf("read clicks: %v", err)
	}
	return total
}

func TestWhatsAppRedirectsToTheConversationAndCountsTheClick(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	branch := addBranch(t, pool, 1, 100)

	rec := clickRedirect(h, fmt.Sprint(branch), browser)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, body %s; want 302", rec.Code, rec.Body)
	}
	want := "https://wa.me/" + number + "?text=Hola%2C%20vi%20tu%20perfil%20en%20MOVO%20y%20quiero%20hacer%20una%20consulta"
	if got := rec.Header().Get("Location"); got != want {
		t.Errorf("Location = %s, want %s", got, want)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store so the next click is counted too", got)
	}
	if strings.Contains(rec.Body.String(), number) {
		t.Error("the response body contains the WhatsApp number")
	}
	if got := clicks(t, pool, branch); got != 1 {
		t.Errorf("clicks = %d, want 1", got)
	}

	clickRedirect(h, fmt.Sprint(branch), browser)
	if got := clicks(t, pool, branch); got != 2 {
		t.Errorf("clicks after a second visit = %d, want 2", got)
	}
}

func TestWhatsAppNamesTheServiceInTheMessageAndCountsItsClick(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	branch := addBranch(t, pool, 1, 100)
	var business, service int64
	if err := pool.QueryRow(t.Context(), `SELECT business_id FROM branches WHERE id = $1`, branch).Scan(&business); err != nil {
		t.Fatalf("business: %v", err)
	}
	if err := pool.QueryRow(t.Context(), `INSERT INTO services (business_id, name) VALUES ($1, 'Yoga') RETURNING id`, business).Scan(&service); err != nil {
		t.Fatalf("service: %v", err)
	}

	rec := clickRedirect(h, fmt.Sprintf("%d?service=%d", branch, service), browser)
	if rec.Code != http.StatusFound || !strings.HasSuffix(rec.Header().Get("Location"), "quiero%20reservar%20Yoga") {
		t.Errorf("status %d, Location %s; want a message that books Yoga", rec.Code, rec.Header().Get("Location"))
	}
	var counted int64
	if err := pool.QueryRow(t.Context(),
		`SELECT COALESCE(sum(count), 0) FROM metric_daily WHERE service_id = $1 AND metric = 'whatsapp_click'`, service).Scan(&counted); err != nil || counted != 1 {
		t.Errorf("clicks on the service = %d, %v; want 1", counted, err)
	}
}

func TestWhatsAppFallsBackToThePlainMessageForAServiceThatDoesNotQualify(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	branch := addBranch(t, pool, 1, 100)

	rec := clickRedirect(h, fmt.Sprintf("%d?service=999999", branch), browser)
	if rec.Code != http.StatusFound || !strings.HasSuffix(rec.Header().Get("Location"), "quiero%20hacer%20una%20consulta") {
		t.Errorf("status %d, Location %s; want the plain message", rec.Code, rec.Header().Get("Location"))
	}
	if got := clicks(t, pool, branch); got != 1 {
		t.Errorf("clicks = %d, want the click counted without the service", got)
	}
}

func TestWhatsAppRedirectsABotButDoesNotCountIt(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	branch := addBranch(t, pool, 1, 100)

	for _, ua := range []string{"WhatsApp/2.24.8.80 A", "Googlebot/2.1", ""} {
		if rec := clickRedirect(h, fmt.Sprint(branch), ua); rec.Code != http.StatusFound {
			t.Errorf("agent %q: status = %d, want 302", ua, rec.Code)
		}
	}
	if got := clicks(t, pool, branch); got != 0 {
		t.Errorf("clicks = %d, want 0 for bots", got)
	}
}

func TestWhatsAppAnswers404ForABranchThatIsNotPublic(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	branch := addBranch(t, pool, 1, 100)
	if _, err := pool.Exec(t.Context(), `UPDATE businesses SET status = 'paused'`); err != nil {
		t.Fatalf("pause: %v", err)
	}

	for name, path := range map[string]string{
		"paused business": fmt.Sprint(branch),
		"unknown branch":  "999999",
		"not a number":    "abc",
		"zero":            "0",
	} {
		rec := clickRedirect(h, path, browser)
		var body httpx.ErrorEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != http.StatusNotFound || body.Error.Code != "branch_not_found" || rec.Header().Get("Location") != "" {
			t.Errorf("%s: status %d, body %s; want 404 branch_not_found and no redirect", name, rec.Code, rec.Body)
		}
	}
	if got := clicks(t, pool, branch); got != 0 {
		t.Errorf("clicks = %d for a hidden branch, want 0", got)
	}
}

func TestWhatsAppRejectsAMalformedService(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	branch := addBranch(t, pool, 1, 100)

	for _, service := range []string{"abc", "0", "-1"} {
		rec := clickRedirect(h, fmt.Sprintf("%d?service=%s", branch, service), browser)
		var body httpx.ErrorEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		details, _ := body.Error.Details.(map[string]any)
		if rec.Code != http.StatusUnprocessableEntity || details["field"] != "service" {
			t.Errorf("service=%s: status %d, body %s; want 422 on service", service, rec.Code, rec.Body)
		}
	}
	if got := clicks(t, pool, branch); got != 0 {
		t.Errorf("clicks = %d for a rejected request, want 0", got)
	}
}
