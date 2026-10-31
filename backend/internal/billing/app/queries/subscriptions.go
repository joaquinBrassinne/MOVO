package queries

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

// ErrSubscriptionFilterInvalid means the status filter is not one of the known values.
var ErrSubscriptionFilterInvalid = errors.New("billing: subscription filter is not valid")

// Filters of ListSubscriptions. Empty means every subscription. Expiring is an active
// subscription that ends within domain.ExpiringSoonDays.
const (
	FilterActive   = "active"
	FilterGrace    = "grace"
	FilterExpired  = "expired"
	FilterExpiring = "expiring"
)

// SubscriptionEntry is a subscription as the admin list shows it.
type SubscriptionEntry struct {
	billing.SubscriptionRow
	Status       domain.Status
	ExpiringSoon bool
	// DaysLeft is how many days remain until paid_until (negative once it passed).
	DaysLeft int
	// ReminderURL is the wa.me link with the reminder for the owner, empty when the
	// number is unknown.
	ReminderURL string
}

// ListSubscriptions answers the "Suscripciones" list and the "por vencer" view.
type ListSubscriptions struct {
	directory billing.SubscriptionDirectory
}

// NewListSubscriptions wires the query.
func NewListSubscriptions(directory billing.SubscriptionDirectory) *ListSubscriptions {
	return &ListSubscriptions{directory: directory}
}

// Handle returns the subscriptions that match filter, the ones ending first on top.
func (q *ListSubscriptions) Handle(ctx context.Context, filter string, today domain.Day) ([]SubscriptionEntry, error) {
	switch filter {
	case "", FilterActive, FilterGrace, FilterExpired, FilterExpiring:
	default:
		return nil, ErrSubscriptionFilterInvalid
	}

	rows, err := q.directory.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}

	entries := make([]SubscriptionEntry, 0, len(rows))
	for _, row := range rows {
		sub := row.Subscription
		entry := SubscriptionEntry{
			SubscriptionRow: row,
			Status:          sub.StatusAt(today),
			ExpiringSoon:    sub.ExpiringSoon(today),
			DaysLeft:        today.DaysUntil(sub.PaidUntil),
			ReminderURL:     ReminderURL(row.OwnerWhatsApp, row.BusinessName, sub.PaidUntil),
		}
		if matches(filter, entry) {
			entries = append(entries, entry)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Subscription.PaidUntil.Before(entries[j].Subscription.PaidUntil)
	})
	return entries, nil
}

func matches(filter string, e SubscriptionEntry) bool {
	switch filter {
	case FilterActive:
		return e.Status == domain.StatusActive
	case FilterGrace:
		return e.Status == domain.StatusGrace
	case FilterExpired:
		return e.Status == domain.StatusExpired
	case FilterExpiring:
		return e.ExpiringSoon
	default:
		return true
	}
}

// ReminderURL builds the wa.me link that opens a chat with the owner and the reminder
// already written. It is empty when there is no number, so the admin sees no dead button.
// The text is for the owner, so it is in Spanish.
func ReminderURL(whatsApp, businessName string, paidUntil domain.Day) string {
	if whatsApp == "" {
		return ""
	}
	t := paidUntil.Time()
	text := fmt.Sprintf("Hola, te escribimos de MOVO: la suscripción de %s vence el %02d/%02d/%d. "+
		"Si el cobro automático de Mercado Pago está al día no tenés que hacer nada; si cambiaste la tarjeta o lo cancelaste, avisanos así no dejás de aparecer en el mapa.",
		businessName, t.Day(), int(t.Month()), t.Year())
	return "https://wa.me/" + whatsApp + "?text=" + url.QueryEscape(text)
}
