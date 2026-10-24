package businesshttp_test

import (
	"net/http"
	"strings"
	"testing"
)

// uploadPhoto does what the browser does: presign, put the file, return the key.
// The put is simulated by marking the object as uploaded in the fake storage.
func uploadPhoto(s *server, token string) string {
	s.t.Helper()

	presigned := s.do(http.MethodPost, "/me/business/images/presign", token, map[string]any{"content_type": "image/jpeg", "size": 2048})
	key, _ := presigned.body["object_key"].(string)
	if presigned.status != http.StatusOK || key == "" {
		s.t.Fatalf("presign = %d %v", presigned.status, presigned.body)
	}
	s.storage.Put(key)
	return key
}

func photoIDs(s *server, token string) []float64 {
	s.t.Helper()

	mine := s.do(http.MethodGet, "/me/business", token, nil)
	images, _ := mine.body["images"].([]any)
	ids := make([]float64, 0, len(images))
	for _, img := range images {
		ids = append(ids, img.(map[string]any)["id"].(float64))
	}
	return ids
}

func TestOwnerUploadsOrdersAndRemovesPhotos(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")

	var registered []reply
	for range 3 {
		got := s.do(http.MethodPost, "/me/business/images", owner, map[string]any{"object_key": uploadPhoto(s, owner)})
		if got.status != http.StatusCreated {
			t.Fatalf("register = %d %v", got.status, got.body)
		}
		registered = append(registered, got)
	}
	if registered[0].body["position"] != float64(0) || registered[2].body["position"] != float64(2) {
		t.Errorf("positions = %v, %v; want them appended in order", registered[0].body["position"], registered[2].body["position"])
	}

	mine := s.do(http.MethodGet, "/me/business", owner, nil)
	images, _ := mine.body["images"].([]any)
	if len(images) != 3 {
		t.Fatalf("images = %v", mine.body["images"])
	}
	if first, _ := images[0].(map[string]any); !strings.HasPrefix(first["url"].(string), "https://") {
		t.Errorf("first photo = %v, want a signed URL to show it", first)
	}

	ids := photoIDs(s, owner)
	if got := s.do(http.MethodPut, "/me/business/images/order", owner, map[string]any{"ids": []float64{ids[2], ids[0], ids[1]}}); got.status != http.StatusNoContent {
		t.Fatalf("reorder = %d %v", got.status, got.body)
	}
	if after := photoIDs(s, owner); after[0] != ids[2] || after[1] != ids[0] || after[2] != ids[1] {
		t.Errorf("order after making the last the cover = %v, want %v", after, []float64{ids[2], ids[0], ids[1]})
	}

	if got := s.do(http.MethodDelete, "/me/business/images/"+itoa(int64(ids[2])), owner, nil); got.status != http.StatusNoContent {
		t.Fatalf("delete the cover = %d", got.status)
	}
	if after := photoIDs(s, owner); len(after) != 2 || after[0] != ids[0] {
		t.Errorf("photos after deleting the cover = %v, want the next one promoted", after)
	}
	if len(s.storage.Deleted()) != 1 {
		t.Errorf("deleted objects = %v, want the removed photo's object gone from storage", s.storage.Deleted())
	}
}

func TestPhotoRegistrationRejections(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")
	other := ownerWithBusiness(s, "other@example.com")
	otherKey := uploadPhoto(s, other)

	key := uploadPhoto(s, owner)
	if got := s.do(http.MethodPost, "/me/business/images", owner, map[string]any{"object_key": key}); got.status != http.StatusCreated {
		t.Fatalf("first registration = %d", got.status)
	}

	tests := []struct {
		name      string
		key       string
		wantCode  int
		wantError string
		wantField string
	}{
		{"registered twice", key, http.StatusConflict, "image_exists", ""},
		{"key of another business", otherKey, http.StatusUnprocessableEntity, "validation_failed", "object_key"},
		{"never uploaded", strings.Replace(key, ".jpg", "x.jpg", 1), http.StatusUnprocessableEntity, "validation_failed", "object_key"},
		{"path traversal", "businesses/../etc/passwd", http.StatusUnprocessableEntity, "validation_failed", "object_key"},
		{"empty", "", http.StatusUnprocessableEntity, "validation_failed", "object_key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.do(http.MethodPost, "/me/business/images", owner, map[string]any{"object_key": tt.key})
			if got.status != tt.wantCode || got.errorCode() != tt.wantError || got.errorField() != tt.wantField {
				t.Fatalf("got %d %q field %q, want %d %q field %q", got.status, got.errorCode(), got.errorField(), tt.wantCode, tt.wantError, tt.wantField)
			}
		})
	}
	if ids := photoIDs(s, owner); len(ids) != 1 {
		t.Errorf("photos = %v, rejected registrations must not be stored", ids)
	}
}

func TestPhotoLimitIsSix(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")

	for i := range 6 {
		if got := s.do(http.MethodPost, "/me/business/images", owner, map[string]any{"object_key": uploadPhoto(s, owner)}); got.status != http.StatusCreated {
			t.Fatalf("photo %d = %d %v", i+1, got.status, got.body)
		}
	}
	got := s.do(http.MethodPost, "/me/business/images", owner, map[string]any{"object_key": uploadPhoto(s, owner)})
	if got.status != http.StatusConflict || got.errorCode() != "image_limit_reached" {
		t.Fatalf("seventh photo = %d %q, want 409 image_limit_reached", got.status, got.errorCode())
	}
}

func TestOwnersCannotTouchEachOthersPhotos(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	alice := ownerWithBusiness(s, "alice@example.com")
	bob := ownerWithBusiness(s, "bob@example.com")
	s.do(http.MethodPost, "/me/business/images", alice, map[string]any{"object_key": uploadPhoto(s, alice)})
	aliceID := photoIDs(s, alice)[0]

	if got := s.do(http.MethodDelete, "/me/business/images/"+itoa(int64(aliceID)), bob, nil); got.status != http.StatusNotFound || got.errorCode() != "image_not_found" {
		t.Errorf("bob deleting Alice's photo = %d %q, want 404 image_not_found", got.status, got.errorCode())
	}
	if got := s.do(http.MethodPut, "/me/business/images/order", bob, map[string]any{"ids": []float64{aliceID}}); got.status != http.StatusUnprocessableEntity {
		t.Errorf("bob reordering Alice's photo = %d, want 422", got.status)
	}
	if got := s.do(http.MethodDelete, "/me/business/images/abc", alice, nil); got.status != http.StatusNotFound {
		t.Errorf("non numeric id = %d, want 404", got.status)
	}
	if ids := photoIDs(s, alice); len(ids) != 1 || len(s.storage.Deleted()) != 0 {
		t.Errorf("Alice's photos = %v, deleted objects = %v; Bob's attempts must change nothing", ids, s.storage.Deleted())
	}
}
