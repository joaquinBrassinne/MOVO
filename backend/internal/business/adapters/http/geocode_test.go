package businesshttp_test

import (
	"net/http"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
)

func TestGeocodeReturnsThePositionOfAnAddress(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	s.geocoder.place = business.Place{Latitude: -34.5875, Longitude: -58.4306, DisplayName: "Honduras 4820, Palermo"}
	owner := ownerWithBusiness(s, "owner@example.com")

	got := s.do(http.MethodPost, "/me/business/geocode", owner, map[string]any{"address": "Honduras 4820", "city": "Buenos Aires"})
	if got.status != http.StatusOK || got.body["latitude"] != -34.5875 || got.body["longitude"] != -58.4306 || got.body["display_name"] != "Honduras 4820, Palermo" {
		t.Fatalf("geocode = %d %v", got.status, got.body)
	}
	if len(s.geocoder.calls) != 1 || s.geocoder.calls[0] != "Honduras 4820, Buenos Aires, Argentina" {
		t.Errorf("geocoder calls = %q", s.geocoder.calls)
	}
}

func TestGeocodeIsOnlyForOwnersWithABusiness(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	_, withoutBusiness := s.user("new@example.com", "business_admin")
	_, admin := s.user("root@example.com", "super_admin")
	body := map[string]any{"address": "Honduras 4820", "city": "Buenos Aires"}

	if got := s.do(http.MethodPost, "/me/business/geocode", "", body); got.status != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", got.status)
	}
	if got := s.do(http.MethodPost, "/me/business/geocode", admin, body); got.status != http.StatusForbidden {
		t.Errorf("super admin = %d, want 403", got.status)
	}
	if got := s.do(http.MethodPost, "/me/business/geocode", withoutBusiness, body); got.status != http.StatusNotFound || got.errorCode() != "business_not_found" {
		t.Errorf("owner without business = %d %q, want 404 business_not_found", got.status, got.errorCode())
	}
	if len(s.geocoder.calls) != 0 {
		t.Errorf("geocoder calls = %q, none of those requests may reach the public service", s.geocoder.calls)
	}
}

func TestGeocodeErrorsHaveStableAnswers(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")
	body := map[string]any{"address": "Honduras 4820", "city": "Buenos Aires"}

	tests := []struct {
		name       string
		err        error
		body       map[string]any
		wantStatus int
		wantCode   string
		wantField  string
		retryAfter bool
	}{
		{"address not found", business.ErrAddressNotFound, body, http.StatusNotFound, "address_not_found", "", false},
		{"geocoder busy", business.ErrGeocoderBusy, body, http.StatusTooManyRequests, "geocoder_busy", "", true},
		{"geocoder down", business.ErrGeocoderUnavailable, body, http.StatusBadGateway, "geocoder_unavailable", "", false},
		{"empty address", nil, map[string]any{"address": " ", "city": "x"}, http.StatusUnprocessableEntity, "validation_failed", "address", false},
		{"empty city", nil, map[string]any{"address": "x", "city": ""}, http.StatusUnprocessableEntity, "validation_failed", "city", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.geocoder.err = tt.err
			got := s.do(http.MethodPost, "/me/business/geocode", owner, tt.body)
			if got.status != tt.wantStatus || got.errorCode() != tt.wantCode || got.errorField() != tt.wantField {
				t.Fatalf("got %d %q field %q, want %d %q field %q", got.status, got.errorCode(), got.errorField(), tt.wantStatus, tt.wantCode, tt.wantField)
			}
			if tt.retryAfter && got.header.Get("Retry-After") == "" {
				t.Error("a busy answer must tell when to retry")
			}
		})
	}
}
