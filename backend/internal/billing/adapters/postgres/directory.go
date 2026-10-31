package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

// List returns the subscription of every business that is not deleted, with the
// business name, its status and the WhatsApp number the owner gave.
func (r *Subscriptions) List(ctx context.Context) ([]billing.SubscriptionRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT s.business_id, s.plan, s.paid_until, COALESCE(s.mp_subscription_ref, ''), COALESCE(s.notes, ''),
       b.name, b.status, b.pilot, b.whatsapp_number
FROM subscriptions s
JOIN businesses b ON b.id = s.business_id
WHERE b.status <> 'deleted'
ORDER BY s.paid_until, s.business_id`)
	if err != nil {
		return nil, fmt.Errorf("postgres: list subscriptions: %w", err)
	}
	defer rows.Close()

	out := []billing.SubscriptionRow{}
	for rows.Next() {
		var (
			row       billing.SubscriptionRow
			plan      string
			paidUntil time.Time
		)
		if err := rows.Scan(&row.Subscription.BusinessID, &plan, &paidUntil, &row.Subscription.MPSubscriptionRef,
			&row.Subscription.Notes, &row.BusinessName, &row.BusinessStatus, &row.Pilot, &row.OwnerWhatsApp); err != nil {
			return nil, fmt.Errorf("postgres: scan subscription: %w", err)
		}
		row.Subscription.Plan = domain.Plan(plan)
		row.Subscription.PaidUntil = domain.DayFromDate(paidUntil)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list subscriptions: %w", err)
	}
	return out, nil
}
