// Package postgres stores subscriptions. It also reads and flips the status of the
// business being confirmed, in the same transaction: confirming a payment must activate
// the business and create its subscription together or not at all, and the two rows live
// in different modules, so the transaction is the only thing that can join them.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

const uniqueViolation = "23505"

// Subscriptions implements billing.SubscriptionRepository.
type Subscriptions struct {
	pool *pgxpool.Pool
}

// NewSubscriptions builds the repository over pool.
func NewSubscriptions(pool *pgxpool.Pool) *Subscriptions {
	return &Subscriptions{pool: pool}
}

// Find returns the subscription of the business.
func (r *Subscriptions) Find(ctx context.Context, businessID int64) (domain.Subscription, error) {
	var (
		sub       domain.Subscription
		plan      string
		paidUntil time.Time
		ref, note *string
	)
	err := r.pool.QueryRow(ctx, `
SELECT business_id, plan, paid_until, mp_subscription_ref, notes
FROM subscriptions WHERE business_id = $1`, businessID).Scan(&sub.BusinessID, &plan, &paidUntil, &ref, &note)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Subscription{}, billing.ErrSubscriptionNotFound
	}
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("postgres: find subscription: %w", err)
	}
	sub.Plan = domain.Plan(plan)
	sub.PaidUntil = domain.DayFromDate(paidUntil)
	sub.MPSubscriptionRef, sub.Notes = deref(ref), deref(note)
	return sub, nil
}

// PlanOf implements billing.PlanResolver: the plan of the subscription, or Starter while
// the business has none yet (an owner fills in the onboarding before the admin confirms
// the payment, and the lowest plan is the safe limit for that stretch).
func (r *Subscriptions) PlanOf(ctx context.Context, businessID int64) (domain.Plan, error) {
	sub, err := r.Find(ctx, businessID)
	if errors.Is(err, billing.ErrSubscriptionNotFound) {
		return domain.PlanStarter, nil
	}
	if err != nil {
		return "", err
	}
	return sub.Plan, nil
}

// Confirm creates the subscription, activates a business that is pending review and
// appends the audit entry, in one transaction. The business row is locked first so two
// admins confirming at once cannot both succeed.
func (r *Subscriptions) Confirm(ctx context.Context, sub domain.Subscription, confirmedBy int64, entry billing.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM businesses WHERE id = $1 FOR UPDATE`, sub.BusinessID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return billing.ErrBusinessNotFound
	}
	if err != nil {
		return fmt.Errorf("postgres: lock business: %w", err)
	}
	if status != "pending_review" && status != "active" {
		return billing.ErrBusinessNotConfirmable
	}

	_, err = tx.Exec(ctx, `
INSERT INTO subscriptions (business_id, plan, paid_until, mp_subscription_ref, confirmed_by, notes)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''))`,
		sub.BusinessID, string(sub.Plan), sub.PaidUntil.Time(), sub.MPSubscriptionRef, confirmedBy, sub.Notes)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return billing.ErrSubscriptionExists
	}
	if err != nil {
		return fmt.Errorf("postgres: insert subscription: %w", err)
	}

	if status == "pending_review" {
		if _, err := tx.Exec(ctx, `UPDATE businesses SET status = 'active', updated_at = now() WHERE id = $1`, sub.BusinessID); err != nil {
			return fmt.Errorf("postgres: activate business: %w", err)
		}
	}
	if err := appendAudit(ctx, tx, entry); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}

// Update stores the plan and paid_until and appends the audit entry. The UPDATE matches
// only while the stored plan and date are still those of previous.
func (r *Subscriptions) Update(ctx context.Context, sub, previous domain.Subscription, entry billing.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
UPDATE subscriptions SET plan = $2, paid_until = $3, updated_at = now()
WHERE business_id = $1 AND plan = $4 AND paid_until = $5`,
		sub.BusinessID, string(sub.Plan), sub.PaidUntil.Time(), string(previous.Plan), previous.PaidUntil.Time())
	if err != nil {
		return fmt.Errorf("postgres: update subscription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM subscriptions WHERE business_id = $1)`, sub.BusinessID).Scan(&exists); err != nil {
			return fmt.Errorf("postgres: check subscription: %w", err)
		}
		if !exists {
			return billing.ErrSubscriptionNotFound
		}
		return billing.ErrSubscriptionConflict
	}
	if err := appendAudit(ctx, tx, entry); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}

func appendAudit(ctx context.Context, tx pgx.Tx, entry billing.AuditEntry) error {
	metadata, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("postgres: encode audit metadata: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO audit_log (actor_id, action, entity, entity_id, metadata) VALUES ($1, $2, 'business', $3, $4)`,
		entry.ActorID, entry.Action, entry.BusinessID, metadata); err != nil {
		return fmt.Errorf("postgres: append audit entry: %w", err)
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
