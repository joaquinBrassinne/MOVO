package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func TestNewBranch(t *testing.T) {
	t.Parallel()

	valid := domain.NewBranchInput{
		Name:       " Centro ",
		Address:    " Córdoba 1234 ",
		City:       " Rosario ",
		Latitude:   -32.95,
		Longitude:  -60.65,
		BookingURL: " https://example.com/reservas ",
	}

	got, err := domain.NewBranch(valid)
	if err != nil {
		t.Fatalf("NewBranch: %v", err)
	}
	if got.Name != "Centro" || got.Address != "Córdoba 1234" || got.City != "Rosario" {
		t.Errorf("text fields = %q/%q/%q, want trimmed values", got.Name, got.Address, got.City)
	}
	if got.BookingURL != "https://example.com/reservas" {
		t.Errorf("booking url = %q, want trimmed value", got.BookingURL)
	}
	if !got.Active {
		t.Error("a new branch must be active")
	}

	empty := valid
	empty.BookingURL = ""
	if _, err := domain.NewBranch(empty); err != nil {
		t.Errorf("booking url is optional, got %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*domain.NewBranchInput)
		want   error
	}{
		{"empty name", func(in *domain.NewBranchInput) { in.Name = " " }, domain.ErrBranchNameInvalid},
		{"name too long", func(in *domain.NewBranchInput) { in.Name = strings.Repeat("a", 101) }, domain.ErrBranchNameInvalid},
		{"empty address", func(in *domain.NewBranchInput) { in.Address = "" }, domain.ErrAddressInvalid},
		{"empty city", func(in *domain.NewBranchInput) { in.City = "" }, domain.ErrCityInvalid},
		{"latitude out of range", func(in *domain.NewBranchInput) { in.Latitude = 91 }, domain.ErrLocationInvalid},
		{"longitude out of range", func(in *domain.NewBranchInput) { in.Longitude = -181 }, domain.ErrLocationInvalid},
		{"missing pin", func(in *domain.NewBranchInput) { in.Latitude, in.Longitude = 0, 0 }, domain.ErrLocationInvalid},
		{"booking url without scheme", func(in *domain.NewBranchInput) { in.BookingURL = "example.com/reservas" }, domain.ErrBookingURLInvalid},
		{"booking url with other scheme", func(in *domain.NewBranchInput) { in.BookingURL = "javascript:alert(1)" }, domain.ErrBookingURLInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := valid
			tt.mutate(&in)
			if _, err := domain.NewBranch(in); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestBranchUpdateKeepsIdentityAndState(t *testing.T) {
	t.Parallel()

	current := domain.Branch{ID: 4, BusinessID: 2, Active: false}
	got, err := current.Update(domain.NewBranchInput{
		Name: "Norte", Address: "Calle 2", City: "Rosario", Latitude: -32.9, Longitude: -60.7,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ID != 4 || got.BusinessID != 2 || got.Active {
		t.Errorf("identity/state = (%d, %d, %v), want (4, 2, false)", got.ID, got.BusinessID, got.Active)
	}
	if got.Name != "Norte" {
		t.Errorf("name = %q, want Norte", got.Name)
	}

	if _, err := current.Update(domain.NewBranchInput{}); !errors.Is(err, domain.ErrBranchNameInvalid) {
		t.Errorf("invalid update error = %v, want ErrBranchNameInvalid", err)
	}
}

func TestBranchReferenceIsOptionalTrimmedAndBounded(t *testing.T) {
	t.Parallel()

	base := domain.NewBranchInput{Name: "Centro", Address: "Calle 1", City: "Rosario", Latitude: -32.95, Longitude: -60.65}

	withRef := base
	withRef.Reference = "  PB, timbre 4B "
	got, err := domain.NewBranch(withRef)
	if err != nil || got.Reference != "PB, timbre 4B" {
		t.Fatalf("branch with reference = %+v, %v", got, err)
	}
	if got, err := domain.NewBranch(base); err != nil || got.Reference != "" {
		t.Errorf("branch without reference = %+v, %v", got, err)
	}

	tooLong := base
	tooLong.Reference = strings.Repeat("a", 101)
	if _, err := domain.NewBranch(tooLong); !errors.Is(err, domain.ErrReferenceInvalid) {
		t.Errorf("101 characters error = %v, want ErrReferenceInvalid", err)
	}
}

func TestBranchActivation(t *testing.T) {
	t.Parallel()

	b := domain.Branch{ID: 1, Active: true, Name: "Centro"}
	off := b.Deactivate()
	if off.Active || off.Name != "Centro" || off.ID != 1 {
		t.Errorf("Deactivate = %+v", off)
	}
	if !off.Activate().Active {
		t.Error("Activate must turn the branch on")
	}
}
