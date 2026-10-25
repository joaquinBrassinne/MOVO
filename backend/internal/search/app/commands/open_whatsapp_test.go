package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
)

type fakeContacts struct {
	contact search.Contact
	err     error
}

func (f fakeContacts) ContactFor(context.Context, int64, int64) (search.Contact, error) {
	return f.contact, f.err
}

var studio = search.Contact{BusinessID: 7, BranchID: 3, BusinessName: "Studio", WhatsApp: "5493571123456"}

func TestWhatsAppLinkPrefillsTheMessage(t *testing.T) {
	t.Parallel()

	withService := studio
	withService.ServiceID, withService.ServiceName = 9, "Yoga & Pilates"

	for name, tc := range map[string]struct {
		contact search.Contact
		want    string
	}{
		"without a service": {studio, "https://wa.me/5493571123456?text=Hola%2C%20vi%20tu%20perfil%20en%20MOVO%20y%20quiero%20hacer%20una%20consulta"},
		"with a service":    {withService, "https://wa.me/5493571123456?text=Hola%2C%20vi%20tu%20perfil%20en%20MOVO%20y%20quiero%20reservar%20Yoga%20%26%20Pilates"},
	} {
		if got := commands.WhatsAppLink(tc.contact); got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", name, got, tc.want)
		}
	}
}

func TestOpenWhatsAppCountsTheClickWithItsBranchAndService(t *testing.T) {
	t.Parallel()

	contact := studio
	contact.ServiceID, contact.ServiceName = 9, "Yoga"
	counter := &fakeCounter{}
	cmd := commands.NewOpenWhatsApp(fakeContacts{contact: contact}, commands.NewRecordMetric(counter, nil), nil)

	link, err := cmd.Handle(t.Context(), 3, 9, chrome)
	if err != nil || link == "" {
		t.Fatalf("Handle: link %q, err %v", link, err)
	}
	want := search.Event{Metric: search.WhatsAppClick, BusinessID: 7, BranchID: 3, ServiceID: 9}
	if len(counter.events) != 1 || counter.events[0] != want {
		t.Errorf("recorded %v, want [%v]", counter.events, want)
	}
}

func TestOpenWhatsAppRedirectsEvenWhenABotClicksOrTheCounterFails(t *testing.T) {
	t.Parallel()

	bot := &fakeCounter{}
	link, err := commands.NewOpenWhatsApp(fakeContacts{contact: studio}, commands.NewRecordMetric(bot, nil), nil).
		Handle(t.Context(), 3, 0, "WhatsApp/2.24.8.80 A")
	if err != nil || link == "" || len(bot.events) != 0 {
		t.Errorf("bot: link %q, err %v, recorded %v; want the link and no count", link, err, bot.events)
	}

	broken := &fakeCounter{err: errors.New("db down")}
	link, err = commands.NewOpenWhatsApp(fakeContacts{contact: studio}, commands.NewRecordMetric(broken, nil), nil).
		Handle(t.Context(), 3, 0, chrome)
	if err != nil || link == "" {
		t.Errorf("counter down: link %q, err %v; want the link anyway", link, err)
	}
}

func TestOpenWhatsAppPassesBranchNotFoundAndCountsNothing(t *testing.T) {
	t.Parallel()

	counter := &fakeCounter{}
	cmd := commands.NewOpenWhatsApp(fakeContacts{err: search.ErrBranchNotFound}, commands.NewRecordMetric(counter, nil), nil)

	if _, err := cmd.Handle(t.Context(), 3, 0, chrome); !errors.Is(err, search.ErrBranchNotFound) {
		t.Errorf("err = %v, want ErrBranchNotFound", err)
	}
	if len(counter.events) != 0 {
		t.Errorf("recorded %v for a missing branch, want nothing", counter.events)
	}
}
