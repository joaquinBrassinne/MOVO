package domain_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func TestNormalizeWhatsApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr error
	}{
		{"already digits", "5491112345678", "5491112345678", nil},
		{"plus and separators", "+54 9 11 1234-5678", "5491112345678", nil},
		{"parentheses and dots", "+54 (9) 11.1234.5678", "5491112345678", nil},
		{"international 00 prefix", "005491112345678", "5491112345678", nil},
		{"surrounding spaces", "  5491112345678 ", "5491112345678", nil},
		{"empty", "", "", domain.ErrWhatsAppInvalid},
		{"letters", "54911abc5678", "", domain.ErrWhatsAppInvalid},
		{"too short", "12345", "", domain.ErrWhatsAppInvalid},
		{"too long", "1234567890123456", "", domain.ErrWhatsAppInvalid},
		{"national format without country code", "011 1234-5678", "", domain.ErrWhatsAppInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.NormalizeWhatsApp(tt.raw)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("number = %q, want %q", got, tt.want)
			}
		})
	}
}
