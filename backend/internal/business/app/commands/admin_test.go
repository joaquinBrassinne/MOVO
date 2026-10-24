package commands_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// fakeAdmin is an in-memory AdminRepository. conflictOnApply makes Apply report
// that someone else changed the business first.
type fakeAdmin struct {
	mu              sync.Mutex
	byID            map[int64]domain.Business
	audit           []business.AuditEntry
	conflictOnApply bool
}

func newFakeAdmin(businesses ...domain.Business) *fakeAdmin {
	f := &fakeAdmin{byID: map[int64]domain.Business{}}
	for _, b := range businesses {
		f.byID[b.ID] = b
	}
	return f
}

func (f *fakeAdmin) FindByID(_ context.Context, id int64) (domain.Business, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	b, ok := f.byID[id]
	if !ok {
		return domain.Business{}, business.ErrBusinessNotFound
	}
	return b, nil
}

func (f *fakeAdmin) Apply(_ context.Context, b domain.Business, expected domain.Status, entry business.AuditEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.conflictOnApply || f.byID[b.ID].Status != expected {
		return business.ErrStatusConflict
	}
	f.byID[b.ID] = b
	f.audit = append(f.audit, entry)
	return nil
}

func adminHandle(t *testing.T, store *fakeAdmin, action commands.AdminAction) (domain.Business, error) {
	t.Helper()

	return commands.NewApplyAdminAction(store).Handle(t.Context(), commands.ApplyAdminActionInput{ActorID: 99, BusinessID: 1, Action: action})
}

func TestAdminActionsMoveTheBusinessThroughItsLifecycleAndAuditEachOne(t *testing.T) {
	t.Parallel()

	store := newFakeAdmin(domain.Business{ID: 1, Status: domain.StatusPendingReview})
	steps := []struct {
		action commands.AdminAction
		want   domain.Status
	}{
		{commands.ActionApprove, domain.StatusActive},
		{commands.ActionPause, domain.StatusPaused},
		{commands.ActionReactivate, domain.StatusActive},
		{commands.ActionDelete, domain.StatusDeleted},
	}
	for _, step := range steps {
		got, err := adminHandle(t, store, step.action)
		if err != nil || got.Status != step.want {
			t.Fatalf("%s = %q, %v; want %q", step.action, got.Status, err, step.want)
		}
	}

	if len(store.audit) != len(steps) {
		t.Fatalf("audit entries = %d, want one per action (%d)", len(store.audit), len(steps))
	}
	first := store.audit[0]
	if first.ActorID != 99 || first.Action != "business.approve" || first.Entity != "business" || first.EntityID != 1 {
		t.Errorf("first entry = %+v", first)
	}
	if first.Metadata["status_from"] != "pending_review" || first.Metadata["status_to"] != "active" {
		t.Errorf("metadata = %v, want the status change", first.Metadata)
	}
}

func TestAdminActionRejectionsWriteNothing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		b      domain.Business
		action commands.AdminAction
		want   error
	}{
		{"approve a draft", domain.Business{ID: 1, Status: domain.StatusDraft}, commands.ActionApprove, domain.ErrInvalidTransition},
		{"pause a pending business", domain.Business{ID: 1, Status: domain.StatusPendingReview}, commands.ActionPause, domain.ErrInvalidTransition},
		{"delete twice", domain.Business{ID: 1, Status: domain.StatusDeleted}, commands.ActionDelete, domain.ErrInvalidTransition},
		{"verify a deleted business", domain.Business{ID: 1, Status: domain.StatusDeleted}, commands.ActionVerify, domain.ErrNotEditable},
		{"mark a deleted business as pilot", domain.Business{ID: 1, Status: domain.StatusDeleted}, commands.ActionMarkPilot, domain.ErrNotEditable},
		{"verify twice", domain.Business{ID: 1, Status: domain.StatusActive, VerifiedBy: 5, VerifiedAt: ptrTime()}, commands.ActionVerify, commands.ErrNoChange},
		{"revoke without seal", domain.Business{ID: 1, Status: domain.StatusActive}, commands.ActionRevokeVerification, commands.ErrNoChange},
		{"unmark a regular business", domain.Business{ID: 1, Status: domain.StatusActive}, commands.ActionUnmarkPilot, commands.ErrNoChange},
		{"unknown action", domain.Business{ID: 1, Status: domain.StatusActive}, "explode", commands.ErrUnknownAdminAction},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAdmin(tt.b)
			if _, err := adminHandle(t, store, tt.action); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if len(store.audit) != 0 || store.byID[1] != tt.b {
				t.Errorf("a rejected action must write nothing: audit %v, business %+v", store.audit, store.byID[1])
			}
		})
	}

	if _, err := adminHandle(t, newFakeAdmin(), commands.ActionApprove); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("missing business error = %v, want ErrBusinessNotFound", err)
	}
}

func TestVerifyRecordsWhoVouchedAndRevokeClearsIt(t *testing.T) {
	t.Parallel()

	store := newFakeAdmin(domain.Business{ID: 1, Status: domain.StatusActive})
	got, err := adminHandle(t, store, commands.ActionVerify)
	if err != nil || !got.Verified() || got.VerifiedBy != 99 {
		t.Fatalf("verify = %+v, %v; want verified by the actor 99", got, err)
	}
	if store.audit[0].Metadata["verified"] != true {
		t.Errorf("metadata = %v, want verified=true", store.audit[0].Metadata)
	}

	got, err = adminHandle(t, store, commands.ActionRevokeVerification)
	if err != nil || got.Verified() || got.VerifiedBy != 0 {
		t.Fatalf("revoke = %+v, %v", got, err)
	}
}

func TestPilotCanBeMarkedAndUnmarked(t *testing.T) {
	t.Parallel()

	store := newFakeAdmin(domain.Business{ID: 1, Status: domain.StatusActive})
	if got, err := adminHandle(t, store, commands.ActionMarkPilot); err != nil || !got.Pilot {
		t.Fatalf("mark = %+v, %v", got, err)
	}
	if got, err := adminHandle(t, store, commands.ActionUnmarkPilot); err != nil || got.Pilot {
		t.Fatalf("unmark = %+v, %v", got, err)
	}
}

func TestConcurrentChangeIsReportedAndNotOverwritten(t *testing.T) {
	t.Parallel()

	store := newFakeAdmin(domain.Business{ID: 1, Status: domain.StatusPendingReview})
	store.conflictOnApply = true
	if _, err := adminHandle(t, store, commands.ActionApprove); !errors.Is(err, business.ErrStatusConflict) {
		t.Fatalf("error = %v, want ErrStatusConflict", err)
	}
	if store.byID[1].Status != domain.StatusPendingReview {
		t.Errorf("status = %q, a conflicting action must not be applied", store.byID[1].Status)
	}
}
