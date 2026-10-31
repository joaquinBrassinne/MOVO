// Package commands holds the billing use cases the super admin runs by hand: Mercado
// Pago is not integrated by API in the MVP, so the admin confirms and registers payments.
package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

// Errors of the commands.
var (
	// ErrPaidUntilInThePast means the initial validity is already over, so the
	// business would be activated and hidden at the same time.
	ErrPaidUntilInThePast = errors.New("billing: paid until must be today or later")
	// ErrNoChange means the subscription already has that plan.
	ErrNoChange = errors.New("billing: subscription already has that plan")
)

// Clock tells the current instant; tests replace it.
type Clock func() time.Time

// ConfirmPaymentInput is the super admin confirming a first payment. The plan comes
// from the admin, never from the owner or the return URL of Mercado Pago.
type ConfirmPaymentInput struct {
	ActorID    int64
	BusinessID int64
	Plan       string
	PaidUntil  domain.Day
	// MPSubscriptionRef and Notes are optional.
	MPSubscriptionRef string
	Notes             string
}

// ConfirmPayment activates a business and creates its subscription in one step, so a
// business can never end up active, unpaid and hidden by accident.
type ConfirmPayment struct {
	repo  billing.SubscriptionRepository
	clock Clock
}

// NewConfirmPayment wires the use case.
func NewConfirmPayment(repo billing.SubscriptionRepository, clock Clock) *ConfirmPayment {
	return &ConfirmPayment{repo: repo, clock: clock}
}

// Handle returns domain.ErrPlanInvalid, domain.ErrPaidUntilInvalid, ErrPaidUntilInThePast,
// domain.ErrRefInvalid, domain.ErrNotesInvalid, billing.ErrBusinessNotFound,
// billing.ErrBusinessNotConfirmable or billing.ErrSubscriptionExists.
func (c *ConfirmPayment) Handle(ctx context.Context, in ConfirmPaymentInput) (domain.Subscription, error) {
	plan, err := domain.ParsePlan(in.Plan)
	if err != nil {
		return domain.Subscription{}, err
	}
	sub, err := domain.NewSubscription(in.BusinessID, plan, in.PaidUntil)
	if err != nil {
		return domain.Subscription{}, err
	}
	if sub.PaidUntil.Before(domain.DayOf(c.clock())) {
		return domain.Subscription{}, ErrPaidUntilInThePast
	}
	if sub, err = sub.WithDetails(in.MPSubscriptionRef, in.Notes); err != nil {
		return domain.Subscription{}, err
	}

	entry := billing.AuditEntry{
		ActorID: in.ActorID, Action: "billing.confirm_payment", BusinessID: in.BusinessID,
		Metadata: map[string]any{
			"plan": string(sub.Plan), "paid_until": sub.PaidUntil.String(),
			"mp_subscription_ref": sub.MPSubscriptionRef,
		},
	}
	if err := c.repo.Confirm(ctx, sub, in.ActorID, entry); err != nil {
		return domain.Subscription{}, fmt.Errorf("confirm payment: %w", err)
	}
	return sub, nil
}

// RegisterPayment records a monthly payment: it extends paid_until by one month.
type RegisterPayment struct {
	repo  billing.SubscriptionRepository
	clock Clock
}

// NewRegisterPayment wires the use case.
func NewRegisterPayment(repo billing.SubscriptionRepository, clock Clock) *RegisterPayment {
	return &RegisterPayment{repo: repo, clock: clock}
}

// Handle returns billing.ErrSubscriptionNotFound or billing.ErrSubscriptionConflict.
func (c *RegisterPayment) Handle(ctx context.Context, actorID, businessID int64) (domain.Subscription, error) {
	current, err := c.repo.Find(ctx, businessID)
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("find subscription: %w", err)
	}
	updated := current.RegisterPayment(domain.DayOf(c.clock()))

	entry := billing.AuditEntry{
		ActorID: actorID, Action: "billing.register_payment", BusinessID: businessID,
		Metadata: map[string]any{
			"paid_until_from": current.PaidUntil.String(), "paid_until_to": updated.PaidUntil.String(),
			"plan": string(current.Plan),
		},
	}
	if err := c.repo.Update(ctx, updated, current, entry); err != nil {
		return domain.Subscription{}, fmt.Errorf("register payment: %w", err)
	}
	return updated, nil
}

// ChangePlan moves a subscription to another plan without touching its dates.
type ChangePlan struct {
	repo billing.SubscriptionRepository
}

// NewChangePlan wires the use case.
func NewChangePlan(repo billing.SubscriptionRepository) *ChangePlan {
	return &ChangePlan{repo: repo}
}

// Handle returns domain.ErrPlanInvalid, billing.ErrSubscriptionNotFound, ErrNoChange or
// billing.ErrSubscriptionConflict.
func (c *ChangePlan) Handle(ctx context.Context, actorID, businessID int64, rawPlan string) (domain.Subscription, error) {
	plan, err := domain.ParsePlan(rawPlan)
	if err != nil {
		return domain.Subscription{}, err
	}
	current, err := c.repo.Find(ctx, businessID)
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("find subscription: %w", err)
	}
	if current.Plan == plan {
		return domain.Subscription{}, ErrNoChange
	}
	updated, err := current.ChangePlan(plan)
	if err != nil {
		return domain.Subscription{}, err
	}

	entry := billing.AuditEntry{
		ActorID: actorID, Action: "billing.change_plan", BusinessID: businessID,
		Metadata: map[string]any{"plan_from": string(current.Plan), "plan_to": string(updated.Plan)},
	}
	if err := c.repo.Update(ctx, updated, current, entry); err != nil {
		return domain.Subscription{}, fmt.Errorf("change plan: %w", err)
	}
	return updated, nil
}
