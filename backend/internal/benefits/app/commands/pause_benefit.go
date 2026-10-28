package commands

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// PauseBenefit lets the super admin pause or resume any benefit (SDD 5.2). The pause is
// apart from the owner's switch, so the owner cannot undo it. The role is checked by the
// HTTP group, like the rest of /admin.
type PauseBenefit struct {
	moderator benefits.BenefitModerator
}

// NewPauseBenefit wires the PauseBenefit use case.
func NewPauseBenefit(moderator benefits.BenefitModerator) *PauseBenefit {
	return &PauseBenefit{moderator: moderator}
}

// Handle sets the pause of the benefit on behalf of actorID and returns the benefit.
func (c *PauseBenefit) Handle(ctx context.Context, actorID, benefitID int64, paused bool) (domain.Benefit, error) {
	b, err := c.moderator.SetAdminPaused(ctx, actorID, benefitID, paused)
	if err != nil {
		return domain.Benefit{}, fmt.Errorf("set admin pause: %w", err)
	}
	return b, nil
}
