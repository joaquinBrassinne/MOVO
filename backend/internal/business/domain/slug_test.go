package domain_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{"plain", "Yoga Center", "yoga-center", nil},
		{"accents and eñe", "Pilates Ñandú Café", "pilates-nandu-cafe", nil},
		{"punctuation collapses", "  Box & Fit -- Rosario!  ", "box-fit-rosario", nil},
		{"digits", "Gym 24 7", "gym-24-7", nil},
		{"only symbols", "!!!", "", domain.ErrSlugInvalid},
		{"empty", "", "", domain.ErrSlugInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.Slugify(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("slug = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSlugWithSuffix(t *testing.T) {
	t.Parallel()

	if got, want := domain.SlugWithSuffix("yoga-center", 2), "yoga-center-2"; got != want {
		t.Errorf("slug = %q, want %q", got, want)
	}
	if got, want := domain.SlugWithSuffix("yoga-center", 1), "yoga-center"; got != want {
		t.Errorf("attempt 1 = %q, want the base slug %q", got, want)
	}
}
