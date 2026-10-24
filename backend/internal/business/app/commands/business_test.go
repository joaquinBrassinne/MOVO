package commands_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func createInput(owner int64, name string) commands.CreateBusinessInput {
	return commands.CreateBusinessInput{
		OwnerID: owner, CategoryID: 1, Name: name, Description: "d", WhatsApp: "+54 9 11 1234-5678",
	}
}

func TestCreateBusinessStoresADraftOwnedByTheCaller(t *testing.T) {
	t.Parallel()

	store := newFakeBusinesses(1)
	got, err := commands.NewCreateBusiness(store).Handle(t.Context(), createInput(7, "Yoga Center"))
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.ID == 0 || got.OwnerID != 7 || got.Slug != "yoga-center" || got.Status != domain.StatusDraft {
		t.Errorf("business = %+v", got)
	}
}

func TestCreateBusinessSuffixesATakenSlug(t *testing.T) {
	t.Parallel()

	store := newFakeBusinesses(1)
	create := commands.NewCreateBusiness(store)
	for _, c := range []struct {
		owner int64
		want  string
	}{{1, "yoga-center"}, {2, "yoga-center-2"}, {3, "yoga-center-3"}} {
		got, err := create.Handle(t.Context(), createInput(c.owner, "Yoga Center"))
		if err != nil {
			t.Fatalf("owner %d: %v", c.owner, err)
		}
		if got.Slug != c.want {
			t.Errorf("owner %d slug = %q, want %q", c.owner, got.Slug, c.want)
		}
	}
}

func TestCreateBusinessGivesUpWhenNoSlugIsFree(t *testing.T) {
	t.Parallel()

	store := newFakeBusinesses(1)
	create := commands.NewCreateBusiness(store)
	for owner := int64(1); owner <= 10; owner++ {
		if _, err := create.Handle(t.Context(), createInput(owner, "Popular")); err != nil {
			t.Fatalf("owner %d: %v", owner, err)
		}
	}
	if _, err := create.Handle(t.Context(), createInput(11, "Popular")); !errors.Is(err, commands.ErrSlugExhausted) {
		t.Fatalf("error = %v, want ErrSlugExhausted", err)
	}
}

func TestCreateBusinessRejections(t *testing.T) {
	t.Parallel()

	store := newFakeBusinesses(1)
	create := commands.NewCreateBusiness(store)
	if _, err := create.Handle(t.Context(), createInput(7, "First")); err != nil {
		t.Fatalf("first business: %v", err)
	}

	tests := []struct {
		name  string
		input commands.CreateBusinessInput
		want  error
	}{
		{"second business for the owner", createInput(7, "Second"), business.ErrOwnerHasBusiness},
		{"unknown category", commands.CreateBusinessInput{OwnerID: 8, CategoryID: 99, Name: "x", WhatsApp: "5491112345678"}, business.ErrCategoryNotFound},
		{"invalid whatsapp", commands.CreateBusinessInput{OwnerID: 9, CategoryID: 1, Name: "x", WhatsApp: "123"}, domain.ErrWhatsAppInvalid},
		{"empty name", commands.CreateBusinessInput{OwnerID: 9, CategoryID: 1, Name: " ", WhatsApp: "5491112345678"}, domain.ErrNameInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := create.Handle(t.Context(), tt.input); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
	if len(store.byID) != 1 {
		t.Errorf("stored businesses = %d, want only the first one", len(store.byID))
	}
}

func TestUpdateBusiness(t *testing.T) {
	t.Parallel()

	store := newFakeBusinesses(1, 2)
	if _, err := commands.NewCreateBusiness(store).Handle(t.Context(), createInput(7, "Old Name")); err != nil {
		t.Fatalf("create: %v", err)
	}
	update := commands.NewUpdateBusiness(store)

	got, err := update.Handle(t.Context(), commands.UpdateBusinessInput{
		OwnerID: 7, CategoryID: 2, Name: "New Name", Description: "x", WhatsApp: "5491198765432",
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Name != "New Name" || got.Slug != "old-name" || got.CategoryID != 2 {
		t.Errorf("business = %+v", got)
	}

	if _, err := update.Handle(t.Context(), commands.UpdateBusinessInput{OwnerID: 99, CategoryID: 1, Name: "x", WhatsApp: "5491198765432"}); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("owner without business error = %v, want ErrBusinessNotFound", err)
	}
	saves := store.saves
	if _, err := update.Handle(t.Context(), commands.UpdateBusinessInput{OwnerID: 7, CategoryID: 2, Name: "", WhatsApp: "5491198765432"}); !errors.Is(err, domain.ErrNameInvalid) {
		t.Errorf("invalid update error = %v, want ErrNameInvalid", err)
	}
	if store.saves != saves {
		t.Error("an invalid update must not reach the store")
	}
}

func TestSubmitBusiness(t *testing.T) {
	t.Parallel()

	store := newFakeBusinesses(1)
	if _, err := commands.NewCreateBusiness(store).Handle(t.Context(), createInput(7, "Studio")); err != nil {
		t.Fatalf("create: %v", err)
	}
	submit := commands.NewSubmitBusiness(store)

	got, err := submit.Handle(t.Context(), commands.SubmitBusinessInput{OwnerID: 7, PaymentReference: " OP-1 "})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Status != domain.StatusPendingReview || got.PaymentReference != "OP-1" {
		t.Errorf("business = %+v", got)
	}

	if _, err := submit.Handle(t.Context(), commands.SubmitBusinessInput{OwnerID: 7}); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("second submit error = %v, want ErrInvalidTransition", err)
	}
	if _, err := submit.Handle(t.Context(), commands.SubmitBusinessInput{OwnerID: 99}); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("owner without business error = %v, want ErrBusinessNotFound", err)
	}
}
