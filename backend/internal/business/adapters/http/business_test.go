package businesshttp_test

import (
	"net/http"
	"strings"
	"testing"
)

func validBusiness(s *server) map[string]any {
	return map[string]any{
		"category_id": s.categoryID("yoga"), "name": "Yoga Center", "description": "Clases para todos", "whatsapp": "+54 9 11 5824-9912",
	}
}

func TestOwnerRoutesRequireAnOwner(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	_, admin := s.user("root@example.com", "super_admin")

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/me/business"},
		{http.MethodPost, "/me/business"},
		{http.MethodPut, "/me/business"},
		{http.MethodPost, "/me/business/submit"},
		{http.MethodPost, "/me/business/images/presign"},
	} {
		if got := s.do(route.method, route.path, "", nil); got.status != http.StatusUnauthorized {
			t.Errorf("%s %s without token = %d, want 401", route.method, route.path, got.status)
		}
		if got := s.do(route.method, route.path, "forged", nil); got.status != http.StatusUnauthorized {
			t.Errorf("%s %s with a forged token = %d, want 401", route.method, route.path, got.status)
		}
		if got := s.do(route.method, route.path, admin, nil); got.status != http.StatusForbidden {
			t.Errorf("%s %s as super admin = %d, want 403", route.method, route.path, got.status)
		}
	}
}

func TestOwnerCreatesAndReadsTheirBusiness(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	_, owner := s.user("owner@example.com", "business_admin")

	if got := s.do(http.MethodGet, "/me/business", owner, nil); got.status != http.StatusNotFound || got.errorCode() != "business_not_found" {
		t.Fatalf("before creating = %d %q, want 404 business_not_found", got.status, got.errorCode())
	}

	created := s.do(http.MethodPost, "/me/business", owner, validBusiness(s))
	if created.status != http.StatusCreated {
		t.Fatalf("create = %d %v", created.status, created.body)
	}
	if created.body["slug"] != "yoga-center" || created.body["status"] != "draft" || created.body["whatsapp"] != "5491158249912" ||
		created.body["pilot"] != false || created.body["verified"] != false {
		t.Errorf("created = %v", created.body)
	}

	mine := s.do(http.MethodGet, "/me/business", owner, nil)
	business, _ := mine.body["business"].(map[string]any)
	branches, ok := mine.body["branches"].([]any)
	if mine.status != http.StatusOK || business["name"] != "Yoga Center" || !ok || len(branches) != 0 {
		t.Fatalf("read = %d %v", mine.status, mine.body)
	}
	if mine.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, owner data must not be cached", mine.header.Get("Cache-Control"))
	}
}

func TestCreateBusinessRejections(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	_, owner := s.user("owner@example.com", "business_admin")
	_, other := s.user("other@example.com", "business_admin")
	if got := s.do(http.MethodPost, "/me/business", owner, validBusiness(s)); got.status != http.StatusCreated {
		t.Fatalf("first create = %d", got.status)
	}

	with := func(key string, value any) map[string]any {
		body := validBusiness(s)
		body[key] = value
		return body
	}
	tests := []struct {
		name      string
		token     string
		body      any
		wantCode  int
		wantError string
		wantField string
	}{
		{"second business for the owner", owner, with("name", "Another"), http.StatusConflict, "business_exists", ""},
		{"invalid whatsapp", other, with("whatsapp", "123"), http.StatusUnprocessableEntity, "validation_failed", "whatsapp"},
		{"empty name", other, with("name", " "), http.StatusUnprocessableEntity, "validation_failed", "name"},
		{"unknown category", other, with("category_id", 9999), http.StatusUnprocessableEntity, "validation_failed", "category_id"},
		{"missing category", other, with("category_id", 0), http.StatusUnprocessableEntity, "validation_failed", "category_id"},
		{"description too long", other, with("description", strings.Repeat("a", 501)), http.StatusUnprocessableEntity, "validation_failed", "description"},
		{"unknown field", other, with("owner_user_id", 1), http.StatusBadRequest, "invalid_request", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.do(http.MethodPost, "/me/business", tt.token, tt.body)
			if got.status != tt.wantCode || got.errorCode() != tt.wantError || got.errorField() != tt.wantField {
				t.Fatalf("got %d %q field %q, want %d %q field %q (%v)", got.status, got.errorCode(), got.errorField(), tt.wantCode, tt.wantError, tt.wantField, got.body)
			}
		})
	}
}

func TestOwnersOnlyEverSeeTheirOwnBusiness(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	_, alice := s.user("alice@example.com", "business_admin")
	_, bob := s.user("bob@example.com", "business_admin")

	body := validBusiness(s)
	body["name"] = "Alice Studio"
	if got := s.do(http.MethodPost, "/me/business", alice, body); got.status != http.StatusCreated {
		t.Fatalf("alice create = %d", got.status)
	}

	if got := s.do(http.MethodGet, "/me/business", bob, nil); got.status != http.StatusNotFound {
		t.Errorf("bob reading = %d, want 404: he has no business and must not see Alice's", got.status)
	}
	if got := s.do(http.MethodPut, "/me/business", bob, validBusiness(s)); got.status != http.StatusNotFound {
		t.Errorf("bob editing = %d, want 404: the business comes from the token, never the body", got.status)
	}
	if got := s.do(http.MethodPost, "/me/business/submit", bob, map[string]any{}); got.status != http.StatusNotFound {
		t.Errorf("bob submitting = %d, want 404", got.status)
	}

	alices := s.do(http.MethodGet, "/me/business", alice, nil)
	if b, _ := alices.body["business"].(map[string]any); b["name"] != "Alice Studio" {
		t.Errorf("alice's business changed: %v", alices.body)
	}
}

func TestOwnerEditsAndSubmitsTheBusiness(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	_, owner := s.user("owner@example.com", "business_admin")
	if got := s.do(http.MethodPost, "/me/business", owner, validBusiness(s)); got.status != http.StatusCreated {
		t.Fatalf("create = %d", got.status)
	}

	edit := validBusiness(s)
	edit["name"] = "Yoga Renamed"
	updated := s.do(http.MethodPut, "/me/business", owner, edit)
	if updated.status != http.StatusOK || updated.body["name"] != "Yoga Renamed" || updated.body["slug"] != "yoga-center" {
		t.Fatalf("update = %d %v, want the new name with the same slug", updated.status, updated.body)
	}

	submitted := s.do(http.MethodPost, "/me/business/submit", owner, map[string]any{"payment_reference": "OP-123"})
	if submitted.status != http.StatusOK || submitted.body["status"] != "pending_review" || submitted.body["payment_reference"] != "OP-123" {
		t.Fatalf("submit = %d %v", submitted.status, submitted.body)
	}

	again := s.do(http.MethodPost, "/me/business/submit", owner, map[string]any{})
	if again.status != http.StatusConflict || again.errorCode() != "invalid_state" {
		t.Errorf("second submit = %d %q, want 409 invalid_state", again.status, again.errorCode())
	}
}

func TestPresignImageUpload(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	_, owner := s.user("owner@example.com", "business_admin")
	created := s.do(http.MethodPost, "/me/business", owner, validBusiness(s))
	id := int64(created.body["id"].(float64))

	got := s.do(http.MethodPost, "/me/business/images/presign", owner, map[string]any{"content_type": "image/png", "size": 4096})
	upload, _ := got.body["upload"].(map[string]any)
	key, _ := got.body["object_key"].(string)
	if got.status != http.StatusOK || upload["method"] != "PUT" || upload["url"] == "" {
		t.Fatalf("presign = %d %v", got.status, got.body)
	}
	if prefix := "businesses/" + itoa(id) + "/"; !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, ".png") {
		t.Errorf("object key = %q, want it inside %s", key, prefix)
	}

	for name, body := range map[string]map[string]any{
		"html":      {"content_type": "text/html", "size": 10},
		"empty":     {"content_type": "image/png", "size": 0},
		"too large": {"content_type": "image/png", "size": 6 << 20},
	} {
		bad := s.do(http.MethodPost, "/me/business/images/presign", owner, body)
		if bad.status != http.StatusUnprocessableEntity || bad.errorCode() != "validation_failed" {
			t.Errorf("%s = %d %q, want 422 validation_failed", name, bad.status, bad.errorCode())
		}
	}
	if len(s.storage.Uploads()) != 1 {
		t.Errorf("presigned uploads = %d, want only the valid one", len(s.storage.Uploads()))
	}
}

func TestCategoriesArePublicAndCacheable(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})

	got := s.do(http.MethodGet, "/categories", "", nil)
	list, _ := got.body["categories"].([]any)
	if got.status != http.StatusOK || len(list) != 7 {
		t.Fatalf("categories = %d %v, want 7 without any token", got.status, got.body)
	}
	if first, _ := list[0].(map[string]any); first["slug"] != "gimnasios" || first["name"] != "Gimnasios" || first["id"] == nil {
		t.Errorf("first category = %v", first)
	}
	if cc := got.header.Get("Cache-Control"); cc != "public, max-age=300" {
		t.Errorf("Cache-Control = %q", cc)
	}
}
