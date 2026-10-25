package commands

import (
	"context"
	"log/slog"
	"net/url"
	"strings"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// OpenWhatsApp counts a click on the WhatsApp button and builds the link that opens
// the conversation with the business (SDD 4.2).
type OpenWhatsApp struct {
	contacts search.ContactFinder
	record   *RecordMetric
	log      *slog.Logger
}

// NewOpenWhatsApp builds the command. A nil logger discards logs.
func NewOpenWhatsApp(contacts search.ContactFinder, record *RecordMetric, log *slog.Logger) *OpenWhatsApp {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &OpenWhatsApp{contacts: contacts, record: record, log: log}
}

// Handle returns the wa.me link for a visible branch, with a message that names the
// service when serviceID qualifies (zero asks for none). It returns
// search.ErrBranchNotFound for a branch that is missing or not public.
//
// A counter that fails never blocks the link: reaching the business is the point of
// the button, and a lost count is not worth a lost customer.
func (o *OpenWhatsApp) Handle(ctx context.Context, branchID, serviceID int64, userAgent string) (string, error) {
	contact, err := o.contacts.ContactFor(ctx, branchID, serviceID)
	if err != nil {
		return "", err
	}

	if _, err := o.record.Handle(ctx, search.Event{
		Metric:     search.WhatsAppClick,
		BusinessID: contact.BusinessID,
		BranchID:   contact.BranchID,
		ServiceID:  contact.ServiceID,
	}, userAgent); err != nil {
		o.log.Warn("whatsapp click not counted", "branch_id", branchID, "error", err)
	}

	return WhatsAppLink(contact), nil
}

// Preview returns the same link as Handle without counting anything. Link
// previewers ask with HEAD to see where a link goes; that is not a click.
func (o *OpenWhatsApp) Preview(ctx context.Context, branchID, serviceID int64) (string, error) {
	contact, err := o.contacts.ContactFor(ctx, branchID, serviceID)
	if err != nil {
		return "", err
	}
	return WhatsAppLink(contact), nil
}

// WhatsAppLink builds https://wa.me/<number>?text=<message>. The number holds only
// digits (the business validates it on save), so it needs no escaping.
func WhatsAppLink(c search.Contact) string {
	message := "Hola, vi tu perfil en MOVO y quiero hacer una consulta"
	if c.ServiceName != "" {
		message = "Hola, vi tu perfil en MOVO y quiero reservar " + c.ServiceName
	}
	// QueryEscape writes a space as "+", which WhatsApp would show literally.
	text := strings.ReplaceAll(url.QueryEscape(message), "+", "%20")
	return "https://wa.me/" + c.WhatsApp + "?text=" + text
}
