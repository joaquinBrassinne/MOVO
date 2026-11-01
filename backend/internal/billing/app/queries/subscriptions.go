package queries

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

// ErrSubscriptionFilterInvalid means the status filter is not one of the known values.
var ErrSubscriptionFilterInvalid = errors.New("billing: subscription filter is not valid")

// ErrPageInvalid means limit or offset is negative or the limit is over MaxSubscriptionPage.
var ErrPageInvalid = errors.New("billing: page is not valid")

// MaxSubscriptionPage is the largest page ListSubscriptions serves.
const MaxSubscriptionPage = 100

// SubscriptionFilter narrows and pages the list. Every field is optional: Status is one of the
// Filter* values, Plan a plan name, Query a case-insensitive part of the business name, and a
// zero Limit means no paging.
type SubscriptionFilter struct {
	Status string
	Plan   string
	Query  string
	Limit  int
	Offset int
}

// SubscriptionPage is the requested slice of the list plus how many entries matched in all.
type SubscriptionPage struct {
	Entries []SubscriptionEntry
	Total   int
}

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
	// PrioritySupport marks the businesses whose plan includes the priority support, so the
	// admin answers them first (SDD 4.2). A pilot gets the support of the Pro plan.
	PrioritySupport bool
}

// ListSubscriptions answers the "Suscripciones" list and the "por vencer" view.
type ListSubscriptions struct {
	directory billing.SubscriptionDirectory
	plans     billing.PlanCatalog
}

// WithPlans tells the query which plans include the priority support. Without it nobody is
// marked.
func (q *ListSubscriptions) WithPlans(plans billing.PlanCatalog) *ListSubscriptions {
	q.plans = plans
	return q
}

// prioritySupport tells whether the row's business has the priority support. Pilot businesses
// are on Pro while the pilot lasts (see pilotplan), whatever their subscription says.
func (q *ListSubscriptions) prioritySupport(row billing.SubscriptionRow) bool {
	if q.plans == nil {
		return false
	}
	plan := row.Subscription.Plan
	if row.Pilot {
		plan = domain.PlanPro
	}
	entitlements, err := q.plans.Entitlements(plan)
	return err == nil && entitlements.PrioritySupport()
}

// NewListSubscriptions wires the query.
func NewListSubscriptions(directory billing.SubscriptionDirectory) *ListSubscriptions {
	return &ListSubscriptions{directory: directory}
}

// Handle returns the page of subscriptions that match filter, the ones ending first on top.
func (q *ListSubscriptions) Handle(ctx context.Context, filter SubscriptionFilter, today domain.Day) (SubscriptionPage, error) {
	switch filter.Status {
	case "", FilterActive, FilterGrace, FilterExpired, FilterExpiring:
	default:
		return SubscriptionPage{}, ErrSubscriptionFilterInvalid
	}
	var plan domain.Plan
	if filter.Plan != "" {
		parsed, err := domain.ParsePlan(filter.Plan)
		if err != nil {
			return SubscriptionPage{}, err
		}
		plan = parsed
	}
	if filter.Limit < 0 || filter.Offset < 0 || filter.Limit > MaxSubscriptionPage {
		return SubscriptionPage{}, ErrPageInvalid
	}

	rows, err := q.directory.List(ctx)
	if err != nil {
		return SubscriptionPage{}, fmt.Errorf("list subscriptions: %w", err)
	}
	needle := strings.ToLower(strings.TrimSpace(filter.Query))

	entries := make([]SubscriptionEntry, 0, len(rows))
	for _, row := range rows {
		sub := row.Subscription
		entry := SubscriptionEntry{
			SubscriptionRow: row,
			Status:          sub.StatusAt(today),
			ExpiringSoon:    sub.ExpiringSoon(today),
			DaysLeft:        today.DaysUntil(sub.PaidUntil),
			ReminderURL:     ReminderURL(row.OwnerWhatsApp, row.BusinessName, sub.PaidUntil),
			PrioritySupport: q.prioritySupport(row),
		}
		if matches(filter.Status, entry) && (plan == "" || sub.Plan == plan) &&
			strings.Contains(strings.ToLower(row.BusinessName), needle) {
			entries = append(entries, entry)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Subscription.PaidUntil.Before(entries[j].Subscription.PaidUntil)
	})
	return paginate(entries, filter.Limit, filter.Offset), nil
}

// paginate cuts the page out of entries. A zero limit keeps everything from offset on.
func paginate(entries []SubscriptionEntry, limit, offset int) SubscriptionPage {
	total := len(entries)
	if offset >= total {
		return SubscriptionPage{Entries: []SubscriptionEntry{}, Total: total}
	}
	end := total
	if limit > 0 && offset+limit < total {
		end = offset + limit
	}
	return SubscriptionPage{Entries: entries[offset:end], Total: total}
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
