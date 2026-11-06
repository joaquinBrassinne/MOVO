package queries

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// ProGate is the one check every Pro metric goes through: it resolves the owner's business
// and refuses, before any data is read, a business whose plan does not include the advanced
// metrics. Each Pro query embeds it, so the rule lives here and nowhere else.
type ProGate struct {
	owners     benefits.OwnerBusinesses
	permission benefits.MetricsPermission
}

// NewProGate wires the gate.
func NewProGate(owners benefits.OwnerBusinesses, permission benefits.MetricsPermission) ProGate {
	return ProGate{owners: owners, permission: permission}
}

// Business returns the id of the owner's business, benefits.ErrPlanUpgradeRequired when its
// plan lacks the Pro metrics, or the error that stops the owner from having a business.
func (g ProGate) Business(ctx context.Context, ownerID int64) (int64, error) {
	businessID, err := g.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return 0, fmt.Errorf("resolve business: %w", err)
	}
	allowed, err := g.permission.AdvancedMetrics(ctx, businessID)
	if err != nil {
		return 0, fmt.Errorf("check metrics permission: %w", err)
	}
	if !allowed {
		return 0, benefits.ErrPlanUpgradeRequired
	}
	return businessID, nil
}
