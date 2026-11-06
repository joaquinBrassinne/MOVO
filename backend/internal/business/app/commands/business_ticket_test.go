package commands_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func withMetrics() fixedEntitlements { return fixedEntitlements{limits: limits{advancedMetrics: true}} }

func ptr(n int) *int { return &n }

// ticketInput is an update that keeps the profile as created and only deals with the ticket.
func ticketInput(created domain.Business, ticket commands.OptionalTicket) commands.UpdateBusinessInput {
	return commands.UpdateBusinessInput{
		OwnerID: 7, CategoryID: created.CategoryID, Name: created.Name, Description: created.Description,
		WhatsApp: created.WhatsApp, AverageTicket: ticket,
	}
}

func newOwnerBusiness(t *testing.T) (*fakeBusinesses, domain.Business) {
	t.Helper()

	store := newFakeBusinesses(1)
	created, err := commands.NewCreateBusiness(store).Handle(t.Context(), createInput(7, "Studio"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return store, created
}

func TestAProBusinessSetsItsAverageTicketAndTheChangeIsAudited(t *testing.T) {
	t.Parallel()

	store, created := newOwnerBusiness(t)
	got, err := commands.NewUpdateBusiness(store, withMetrics()).Handle(t.Context(),
		ticketInput(created, commands.OptionalTicket{Set: true, Value: ptr(15000)}))
	if err != nil || got.AverageTicketARS == nil || *got.AverageTicketARS != 15000 {
		t.Fatalf("Handle = %+v, %v; want the ticket 15000", got, err)
	}
	if len(store.audit) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(store.audit))
	}
	changes, _ := store.audit[0].Metadata["changes"].(map[string]map[string]any)
	if len(changes) != 1 || changes["average_ticket_ars"]["from"] != nil || changes["average_ticket_ars"]["to"] != 15000 {
		t.Errorf("changes = %v, want only average_ticket_ars from nil to 15000", changes)
	}
}

func TestAStarterBusinessCannotSetAnAverageTicketAndNothingIsWritten(t *testing.T) {
	t.Parallel()

	store, created := newOwnerBusiness(t)
	saves := store.saves
	_, err := commands.NewUpdateBusiness(store, unlimited()).Handle(t.Context(),
		ticketInput(created, commands.OptionalTicket{Set: true, Value: ptr(15000)}))
	if !errors.Is(err, business.ErrPlanUpgradeRequired) {
		t.Fatalf("err = %v, want ErrPlanUpgradeRequired", err)
	}
	if store.saves != saves || len(store.audit) != 0 {
		t.Errorf("saves %d -> %d, audit %d; want nothing written", saves, store.saves, len(store.audit))
	}
}

func TestADowngradedBusinessKeepsOrClearsItsTicketButCannotChangeIt(t *testing.T) {
	t.Parallel()

	store, created := newOwnerBusiness(t)
	if _, err := commands.NewUpdateBusiness(store, withMetrics()).Handle(t.Context(),
		ticketInput(created, commands.OptionalTicket{Set: true, Value: ptr(15000)})); err != nil {
		t.Fatalf("set while Pro: %v", err)
	}
	starter := commands.NewUpdateBusiness(store, unlimited())

	// Leaving the field out keeps the stored ticket; sending the same value is no change.
	if got, err := starter.Handle(t.Context(), ticketInput(created, commands.OptionalTicket{})); err != nil || got.AverageTicketARS == nil || *got.AverageTicketARS != 15000 {
		t.Errorf("field absent: %+v, %v; want the ticket kept", got, err)
	}
	if _, err := starter.Handle(t.Context(), ticketInput(created, commands.OptionalTicket{Set: true, Value: ptr(15000)})); err != nil {
		t.Errorf("same value: %v, want no error", err)
	}
	// A new value is refused, removing it is allowed.
	if _, err := starter.Handle(t.Context(), ticketInput(created, commands.OptionalTicket{Set: true, Value: ptr(20000)})); !errors.Is(err, business.ErrPlanUpgradeRequired) {
		t.Errorf("new value: err = %v, want ErrPlanUpgradeRequired", err)
	}
	got, err := starter.Handle(t.Context(), ticketInput(created, commands.OptionalTicket{Set: true}))
	if err != nil || got.AverageTicketARS != nil {
		t.Errorf("clearing: %+v, %v; want the ticket removed", got, err)
	}
}

func TestAnAverageTicketMustBeAWholeNumberAboveZero(t *testing.T) {
	t.Parallel()

	store, created := newOwnerBusiness(t)
	update := commands.NewUpdateBusiness(store, withMetrics())
	for _, n := range []int{0, -1, -15000, 2_147_483_648} {
		if _, err := update.Handle(t.Context(), ticketInput(created, commands.OptionalTicket{Set: true, Value: ptr(n)})); !errors.Is(err, domain.ErrAverageTicketInvalid) {
			t.Errorf("ticket %d: err = %v, want ErrAverageTicketInvalid", n, err)
		}
	}
	if _, err := update.Handle(t.Context(), ticketInput(created, commands.OptionalTicket{Set: true, Value: ptr(1)})); err != nil {
		t.Errorf("ticket 1: %v, want it accepted", err)
	}
}
