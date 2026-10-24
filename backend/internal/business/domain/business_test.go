package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func TestNewBusiness(t *testing.T) {
	t.Parallel()

	valid := domain.NewBusinessInput{
		OwnerID:     7,
		CategoryID:  3,
		Name:        "  Yoga Center ",
		Description: " Clases para todos ",
		WhatsApp:    "+54 9 11 1234-5678",
	}

	got, err := domain.NewBusiness(valid)
	if err != nil {
		t.Fatalf("NewBusiness: %v", err)
	}
	if got.Name != "Yoga Center" || got.Description != "Clases para todos" {
		t.Errorf("name/description = %q/%q, want trimmed values", got.Name, got.Description)
	}
	if got.Slug != "yoga-center" {
		t.Errorf("slug = %q, want yoga-center", got.Slug)
	}
	if got.WhatsApp != "5491112345678" {
		t.Errorf("whatsapp = %q, want digits only", got.WhatsApp)
	}
	if got.Status != domain.StatusDraft {
		t.Errorf("status = %q, want draft", got.Status)
	}
	if got.Pilot || got.Verified() {
		t.Errorf("a new business must be neither pilot nor verified")
	}

	tests := []struct {
		name   string
		mutate func(*domain.NewBusinessInput)
		want   error
	}{
		{"empty name", func(in *domain.NewBusinessInput) { in.Name = "  " }, domain.ErrNameInvalid},
		{"name too long", func(in *domain.NewBusinessInput) { in.Name = strings.Repeat("a", 101) }, domain.ErrNameInvalid},
		{"name without letters or digits", func(in *domain.NewBusinessInput) { in.Name = "***" }, domain.ErrSlugInvalid},
		{"description too long", func(in *domain.NewBusinessInput) { in.Description = strings.Repeat("a", 1001) }, domain.ErrDescriptionInvalid},
		{"invalid whatsapp", func(in *domain.NewBusinessInput) { in.WhatsApp = "123" }, domain.ErrWhatsAppInvalid},
		{"missing category", func(in *domain.NewBusinessInput) { in.CategoryID = 0 }, domain.ErrCategoryRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := valid
			tt.mutate(&in)
			if _, err := domain.NewBusiness(in); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestStatusTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		from   domain.Status
		action func(domain.Business) (domain.Business, error)
		want   domain.Status
		err    error
	}{
		{"submit a draft", domain.StatusDraft, domain.Business.Submit, domain.StatusPendingReview, nil},
		{"submit twice", domain.StatusPendingReview, domain.Business.Submit, "", domain.ErrInvalidTransition},
		{"approve pending", domain.StatusPendingReview, domain.Business.Approve, domain.StatusActive, nil},
		{"approve a draft", domain.StatusDraft, domain.Business.Approve, "", domain.ErrInvalidTransition},
		{"pause active", domain.StatusActive, domain.Business.Pause, domain.StatusPaused, nil},
		{"pause pending", domain.StatusPendingReview, domain.Business.Pause, "", domain.ErrInvalidTransition},
		{"reactivate paused", domain.StatusPaused, domain.Business.Reactivate, domain.StatusActive, nil},
		{"reactivate active", domain.StatusActive, domain.Business.Reactivate, "", domain.ErrInvalidTransition},
		{"delete active", domain.StatusActive, domain.Business.Delete, domain.StatusDeleted, nil},
		{"delete draft", domain.StatusDraft, domain.Business.Delete, domain.StatusDeleted, nil},
		{"delete twice", domain.StatusDeleted, domain.Business.Delete, "", domain.ErrInvalidTransition},
		{"resurrect a deleted business", domain.StatusDeleted, domain.Business.Reactivate, "", domain.ErrInvalidTransition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.action(domain.Business{Status: tt.from})
			if !errors.Is(err, tt.err) {
				t.Fatalf("error = %v, want %v", err, tt.err)
			}
			if tt.err == nil && got.Status != tt.want {
				t.Errorf("status = %q, want %q", got.Status, tt.want)
			}
		})
	}
}

func TestVerification(t *testing.T) {
	t.Parallel()

	b := domain.Business{Status: domain.StatusActive}
	b = b.MarkVerified(9)
	if !b.Verified() || b.VerifiedBy != 9 {
		t.Fatalf("after MarkVerified: verified=%v by=%d", b.Verified(), b.VerifiedBy)
	}
	b = b.RevokeVerification()
	if b.Verified() || b.VerifiedBy != 0 {
		t.Errorf("after RevokeVerification: verified=%v by=%d", b.Verified(), b.VerifiedBy)
	}
}

func TestEditableByOwnerOnlyWhileNotDeleted(t *testing.T) {
	t.Parallel()

	for _, s := range []domain.Status{domain.StatusDraft, domain.StatusPendingReview, domain.StatusActive, domain.StatusPaused} {
		if !(domain.Business{Status: s}).Editable() {
			t.Errorf("status %q must be editable", s)
		}
	}
	if (domain.Business{Status: domain.StatusDeleted}).Editable() {
		t.Error("a deleted business must not be editable")
	}
}
