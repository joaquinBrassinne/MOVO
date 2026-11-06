package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func TestValidateImageKeyAcceptsOnlyKeysInsideTheBusiness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want error
	}{
		{"jpg", "businesses/7/abc123.jpg", nil},
		{"png", "businesses/7/abc123.png", nil},
		{"webp", "businesses/7/abc123.webp", nil},
		{"another business", "businesses/8/abc123.jpg", domain.ErrImageKeyInvalid},
		{"id prefix of another business", "businesses/70/abc123.jpg", domain.ErrImageKeyInvalid},
		{"outside the businesses folder", "avatars/7/abc123.jpg", domain.ErrImageKeyInvalid},
		{"nested folder", "businesses/7/sub/abc123.jpg", domain.ErrImageKeyInvalid},
		{"parent traversal", "businesses/7/../8/abc.jpg", domain.ErrImageKeyInvalid},
		{"html extension", "businesses/7/abc123.html", domain.ErrImageKeyInvalid},
		{"no extension", "businesses/7/abc123", domain.ErrImageKeyInvalid},
		{"empty name", "businesses/7/.jpg", domain.ErrImageKeyInvalid},
		{"name with spaces", "businesses/7/a b.jpg", domain.ErrImageKeyInvalid},
		{"name too long", "businesses/7/" + strings.Repeat("a", 101) + ".jpg", domain.ErrImageKeyInvalid},
		{"empty", "", domain.ErrImageKeyInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := domain.ValidateImageKey(7, tt.key); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}
