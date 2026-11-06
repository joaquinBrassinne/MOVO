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
		{"description too long", func(in *domain.NewBusinessInput) { in.Description = strings.Repeat("a", 501) }, domain.ErrDescriptionInvalid},
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

func TestUpdateProfileKeepsSlugAndStatus(t *testing.T) {
	t.Parallel()

	current := domain.Business{ID: 5, Slug: "old-name", Status: domain.StatusActive, Pilot: true}
	got, err := current.UpdateProfile(domain.ProfileInput{
		CategoryID:  2,
		Name:        " New Name ",
		Description: "text",
		WhatsApp:    "+54 9 11 1234-5678",
	})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if got.Slug != "old-name" {
		t.Errorf("slug = %q, want the URL to stay stable", got.Slug)
	}
	if got.ID != 5 || got.Status != domain.StatusActive || !got.Pilot {
		t.Errorf("identity/state changed: id=%d status=%q pilot=%v", got.ID, got.Status, got.Pilot)
	}
	if got.Name != "New Name" || got.CategoryID != 2 || got.WhatsApp != "5491112345678" {
		t.Errorf("profile = %q/%d/%q", got.Name, got.CategoryID, got.WhatsApp)
	}

	if _, err := current.UpdateProfile(domain.ProfileInput{CategoryID: 2, Name: "", WhatsApp: "5491112345678"}); !errors.Is(err, domain.ErrNameInvalid) {
		t.Errorf("empty name error = %v, want ErrNameInvalid", err)
	}
	if _, err := (domain.Business{Status: domain.StatusDeleted}).UpdateProfile(domain.ProfileInput{CategoryID: 2, Name: "x", WhatsApp: "5491112345678"}); !errors.Is(err, domain.ErrNotEditable) {
		t.Errorf("deleted business error = %v, want ErrNotEditable", err)
	}
}

func TestDeclarePayer(t *testing.T) {
	t.Parallel()

	got, err := (domain.Business{}).DeclarePayer("  ana@example.com ", "  Ana Pérez ")
	if err != nil || got.MPPayerEmail != "ana@example.com" || got.MPPayerName != "Ana Pérez" {
		t.Fatalf("DeclarePayer = %q, %q, %v; want the trimmed values", got.MPPayerEmail, got.MPPayerName, err)
	}

	for _, email := range []string{"", "  ", "ana", "ana@", "@example.com", "ana@example", "Ana <ana@example.com>", "a b@example.com", strings.Repeat("a", 250) + "@example.com"} {
		if _, err := (domain.Business{}).DeclarePayer(email, "Ana Pérez"); !errors.Is(err, domain.ErrPayerEmailInvalid) {
			t.Errorf("email %q error = %v, want ErrPayerEmailInvalid", email, err)
		}
	}
	for _, name := range []string{"", "   ", "Al", " Al ", strings.Repeat("a", 101)} {
		if _, err := (domain.Business{}).DeclarePayer("ana@example.com", name); !errors.Is(err, domain.ErrPayerNameInvalid) {
			t.Errorf("name %q error = %v, want ErrPayerNameInvalid", name, err)
		}
	}
	if _, err := (domain.Business{}).DeclarePayer("ana@example.com", "Ana"); err != nil {
		t.Errorf("a three-letter name error = %v, want none", err)
	}
}

func TestRecordReturnParamsKeepsOnlyUsableHints(t *testing.T) {
	t.Parallel()

	got := (domain.Business{}).RecordReturnParams(map[string]string{
		"plan":                  " pro ",
		"preapproval_plan":      "2c93",
		"":                      "no key",
		"empty":                 "  ",
		"long":                  strings.Repeat("v", 201),
		strings.Repeat("k", 41): "long key",
	})
	if len(got.MPReturnParams) != 2 || got.PlanHint() != "pro" || got.MPReturnParams["preapproval_plan"] != "2c93" {
		t.Errorf("params = %v, want only plan and preapproval_plan", got.MPReturnParams)
	}

	many := map[string]string{}
	for i := range 25 {
		many[strings.Repeat("k", i+1)] = "v"
	}
	if n := len((domain.Business{}).RecordReturnParams(many).MPReturnParams); n != 10 {
		t.Errorf("kept %d params, want at most 10", n)
	}

	none := (domain.Business{}).RecordReturnParams(nil)
	if none.MPReturnParams != nil || none.PlanHint() != "" {
		t.Errorf("no params = %v / %q, want nothing", none.MPReturnParams, none.PlanHint())
	}
}

func TestSetPilot(t *testing.T) {
	t.Parallel()

	b, err := (domain.Business{Status: domain.StatusActive}).SetPilot(true)
	if err != nil || !b.Pilot {
		t.Fatalf("SetPilot(true) = %+v, %v", b, err)
	}
	if b, err = b.SetPilot(false); err != nil || b.Pilot {
		t.Fatalf("SetPilot(false) = %+v, %v", b, err)
	}
	if _, err := (domain.Business{Status: domain.StatusDeleted}).SetPilot(true); !errors.Is(err, domain.ErrNotEditable) {
		t.Errorf("deleted business error = %v, want ErrNotEditable", err)
	}
}

func TestDescriptionLimitIsFiveHundredCharacters(t *testing.T) {
	t.Parallel()

	in := domain.NewBusinessInput{CategoryID: 1, Name: "Studio", WhatsApp: "5491112345678"}

	in.Description = strings.Repeat("á", 500)
	if _, err := domain.NewBusiness(in); err != nil {
		t.Errorf("500 characters must be accepted (they are counted as characters, not bytes): %v", err)
	}
	in.Description = strings.Repeat("á", 501)
	if _, err := domain.NewBusiness(in); !errors.Is(err, domain.ErrDescriptionInvalid) {
		t.Errorf("501 characters error = %v, want ErrDescriptionInvalid", err)
	}
}
