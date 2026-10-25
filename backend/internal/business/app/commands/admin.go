package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// Admin errors.
var (
	// ErrUnknownAdminAction means the action is not one of the AdminAction values.
	ErrUnknownAdminAction = errors.New("business: unknown admin action")
	// ErrNoChange means the business is already in the requested state.
	ErrNoChange = errors.New("business: business is already in that state")
)

// AdminAction is something a super admin can do to a business.
type AdminAction string

// The actions of the super admin panel.
const (
	ActionApprove            AdminAction = "approve"
	ActionPause              AdminAction = "pause"
	ActionReactivate         AdminAction = "reactivate"
	ActionDelete             AdminAction = "delete"
	ActionVerify             AdminAction = "verify"
	ActionRevokeVerification AdminAction = "revoke_verification"
	ActionMarkPilot          AdminAction = "mark_pilot"
	ActionUnmarkPilot        AdminAction = "unmark_pilot"
)

// ApplyAdminActionInput is one super admin decision. The caller must already have
// checked the super_admin role; ActorID comes from the session.
type ApplyAdminActionInput struct {
	ActorID    int64
	BusinessID int64
	Action     AdminAction
}

// ApplyAdminAction changes a business on behalf of a super admin and records it
// in the audit log, in the same transaction.
type ApplyAdminAction struct {
	admin business.AdminRepository
}

// NewApplyAdminAction wires the ApplyAdminAction use case.
func NewApplyAdminAction(admin business.AdminRepository) *ApplyAdminAction {
	return &ApplyAdminAction{admin: admin}
}

// Handle applies the action. It returns business.ErrBusinessNotFound,
// domain.ErrInvalidTransition when the status does not allow it,
// domain.ErrNotEditable for a deleted business, ErrNoChange when nothing would
// change and business.ErrStatusConflict when someone else changed the business first.
func (c *ApplyAdminAction) Handle(ctx context.Context, in ApplyAdminActionInput) (domain.Business, error) {
	current, err := c.admin.FindByID(ctx, in.BusinessID)
	if err != nil {
		return domain.Business{}, fmt.Errorf("find business: %w", err)
	}

	updated, err := apply(current, in.Action, in.ActorID)
	if err != nil {
		return domain.Business{}, err
	}

	entry := business.AuditEntry{
		ActorID:  in.ActorID,
		Action:   "business." + string(in.Action),
		Entity:   "business",
		EntityID: current.ID,
		Metadata: map[string]any{
			"status_from": string(current.Status),
			"status_to":   string(updated.Status),
			"pilot":       updated.Pilot,
			"verified":    updated.Verified(),
		},
	}
	if err := c.admin.Apply(ctx, updated, current, entry); err != nil {
		return domain.Business{}, fmt.Errorf("apply %s: %w", in.Action, err)
	}
	return updated, nil
}

func apply(b domain.Business, action AdminAction, actor int64) (domain.Business, error) {
	switch action {
	case ActionApprove:
		return b.Approve()
	case ActionPause:
		return b.Pause()
	case ActionReactivate:
		return b.Reactivate()
	case ActionDelete:
		return b.Delete()
	case ActionVerify:
		return changeIf(b, !b.Verified(), func(b domain.Business) domain.Business { return b.MarkVerified(actor) })
	case ActionRevokeVerification:
		return changeIf(b, b.Verified(), domain.Business.RevokeVerification)
	case ActionMarkPilot:
		return changeIf(b, !b.Pilot, func(b domain.Business) domain.Business { return mustPilot(b, true) })
	case ActionUnmarkPilot:
		return changeIf(b, b.Pilot, func(b domain.Business) domain.Business { return mustPilot(b, false) })
	default:
		return domain.Business{}, ErrUnknownAdminAction
	}
}

// mustPilot sets the flag on a business changeIf already proved editable, so
// SetPilot cannot fail here.
func mustPilot(b domain.Business, pilot bool) domain.Business {
	updated, err := b.SetPilot(pilot)
	if err != nil {
		return b
	}
	return updated
}

// changeIf runs change only when it would alter the business and the business is
// not deleted.
func changeIf(b domain.Business, wouldChange bool, change func(domain.Business) domain.Business) (domain.Business, error) {
	if !b.Editable() {
		return domain.Business{}, domain.ErrNotEditable
	}
	if !wouldChange {
		return domain.Business{}, ErrNoChange
	}
	return change(b), nil
}
