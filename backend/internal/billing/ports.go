// Package billing is the module that knows the plans and what each one allows.
// This file declares the ports the application layer consumes; adapters
// implement them.
package billing

import (
	"context"
	"errors"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

// ErrUnknownPlan means the catalog has no entry for the plan.
var ErrUnknownPlan = errors.New("billing: plan is not in the catalog")

// PlanInfo is a plan as shown to owners.
type PlanInfo struct {
	Plan  domain.Plan
	Price int
	// CheckoutURL is where the owner pays: the Mercado Pago link of the plan.
	CheckoutURL  string
	Entitlements domain.Entitlements
}

// PlanCatalog tells what each plan offers. Today the values come from
// configuration; a database-backed catalog can replace it without touching the
// domain.
type PlanCatalog interface {
	// Entitlements returns the limits of the plan or ErrUnknownPlan.
	Entitlements(plan domain.Plan) (domain.Entitlements, error)
	// Plans lists every plan, cheapest first.
	Plans() []PlanInfo
}

// PlanResolver tells which plan a business is on. Subscriptions arrive in
// phase 5; until then a fixed resolver answers for everyone.
type PlanResolver interface {
	PlanOf(ctx context.Context, businessID int64) (domain.Plan, error)
}

// Errors of the subscription use cases.
var (
	// ErrBusinessNotFound means there is no business with that id.
	ErrBusinessNotFound = errors.New("billing: business not found")
	// ErrBusinessNotConfirmable means the business is not waiting for a payment
	// confirmation: only a business pending review or already active can get one.
	ErrBusinessNotConfirmable = errors.New("billing: business is not pending review or active")
	// ErrSubscriptionExists means the business already has a subscription: register a
	// payment or change the plan instead of confirming again.
	ErrSubscriptionExists = errors.New("billing: business already has a subscription")
	// ErrSubscriptionNotFound means the business has no subscription yet.
	ErrSubscriptionNotFound = errors.New("billing: subscription not found")
	// ErrSubscriptionConflict means someone else changed the subscription first.
	ErrSubscriptionConflict = errors.New("billing: subscription was changed by someone else")
)

// AuditEntry is one line of the audit trail. It is written about the business
// (entity "business"), so it shows in the business history of the super admin.
type AuditEntry struct {
	ActorID    int64
	Action     string
	BusinessID int64
	Metadata   map[string]any
}

// SubscriptionRepository stores the subscription of each business. Every write comes
// with its audit entry and happens in one transaction.
type SubscriptionRepository interface {
	// Find returns ErrSubscriptionNotFound when the business has none.
	Find(ctx context.Context, businessID int64) (domain.Subscription, error)
	// Confirm creates the subscription and, if the business is pending review,
	// activates it, all or nothing. It returns ErrBusinessNotFound,
	// ErrBusinessNotConfirmable or ErrSubscriptionExists and writes nothing.
	Confirm(ctx context.Context, sub domain.Subscription, confirmedBy int64, entry AuditEntry) error
	// Update stores the plan and paid_until of sub, provided the stored ones are still
	// those of previous; otherwise it returns ErrSubscriptionConflict.
	Update(ctx context.Context, sub, previous domain.Subscription, entry AuditEntry) error
	// ChangePlan is Update for a plan change that also applies the limits of the new
	// plan (SDD 4.2): under one lock on the business, what exceeds ent is switched off,
	// oldest kept, nothing deleted, and an alert tells the owner. A pilot keeps the Pro
	// limits whatever its subscription says, so nothing is switched off for it. The audit
	// entry gets the switched-off ids in its metadata.
	ChangePlan(ctx context.Context, sub, previous domain.Subscription, ent domain.Entitlements, entry AuditEntry) (Downgrade, error)
}

// Item is a branch or a service by id and name.
type Item struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Downgrade is what a plan change switched off.
type Downgrade struct {
	Branches []Item
	Services []Item
}

// Empty reports whether nothing was switched off.
func (d Downgrade) Empty() bool { return len(d.Branches) == 0 && len(d.Services) == 0 }

// Alert is a notice for the owner of a business.
type Alert struct {
	ID        int64
	Type      string
	Payload   map[string]any
	CreatedAt time.Time
}

// ErrAlertNotFound means the owner has no such alert (or it is not theirs).
var ErrAlertNotFound = errors.New("billing: alert not found")

// OwnerAlerts is what the owner reads. Both methods are scoped by the owner id taken from
// the token, so an owner can never see or dismiss the alert of another business.
type OwnerAlerts interface {
	// Unread returns the unread alerts of the owner's business, newest first; none
	// (not an error) when the owner has no business.
	Unread(ctx context.Context, ownerID int64) ([]Alert, error)
	// MarkRead dismisses one alert and returns ErrAlertNotFound when it is not the
	// owner's. Dismissing an already read alert is not an error.
	MarkRead(ctx context.Context, ownerID, alertID int64) error
}

// SubscriptionRow is one subscription with what the super admin needs to act on it:
// who the business is and where to reach the owner.
type SubscriptionRow struct {
	Subscription   domain.Subscription
	BusinessName   string
	BusinessStatus string
	Pilot          bool
	// OwnerWhatsApp is the number the owner gave, digits only.
	OwnerWhatsApp string
}

// SubscriptionDirectory lists every subscription. The MVP has a few dozen businesses
// (SDD risks table), so the list is read whole and filtered by the domain rules,
// instead of repeating them in SQL.
type SubscriptionDirectory interface {
	List(ctx context.Context) ([]SubscriptionRow, error)
}

// ErrOwnerHasNoBusiness means the owner has not created a business, so there is no plan.
var ErrOwnerHasNoBusiness = errors.New("billing: owner has no business")

// OwnerPlans tells which plan an owner's business is on. The composition root builds
// it from the business module and the PlanResolver, so billing never reads businesses.
type OwnerPlans interface {
	// PlanOfOwner returns ErrOwnerHasNoBusiness when the owner has no business.
	PlanOfOwner(ctx context.Context, ownerID int64) (PlanInfo, error)
}
