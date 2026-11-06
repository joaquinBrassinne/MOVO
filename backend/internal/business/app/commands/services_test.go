package commands_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func addService(t *testing.T, add *commands.AddService, owner int64, name string) domain.Service {
	t.Helper()

	got, err := add.Handle(t.Context(), commands.AddServiceInput{OwnerID: owner, Service: domain.ServiceInput{Name: name}})
	if err != nil {
		t.Fatalf("add service %q: %v", name, err)
	}
	return got
}

func TestAddServiceAttachesItToTheCallersBusiness(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	services := newFakeServices()
	owned, _ := businesses.FindByOwner(t.Context(), 7)
	branch, err := commands.NewAddBranch(businesses, branches, unlimited()).Handle(t.Context(), commands.AddBranchInput{OwnerID: 7, Branch: validBranch})
	if err != nil {
		t.Fatalf("add branch: %v", err)
	}
	add := commands.NewAddService(businesses, branches, services, unlimited())

	got, err := add.Handle(t.Context(), commands.AddServiceInput{OwnerID: 7, Service: domain.ServiceInput{Name: "Yoga", BranchID: branch.ID}})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.ID == 0 || got.BusinessID != owned.ID || got.BranchID != branch.ID || !got.Active {
		t.Errorf("service = %+v", got)
	}
}

func TestAddServiceRejections(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	services := newFakeServices()
	add := commands.NewAddService(businesses, branches, services, unlimited())
	theirBranch, err := commands.NewAddBranch(businesses, branches, unlimited()).Handle(t.Context(), commands.AddBranchInput{OwnerID: 8, Branch: validBranch})
	if err != nil {
		t.Fatalf("add branch: %v", err)
	}

	tests := []struct {
		name  string
		input commands.AddServiceInput
		want  error
	}{
		{"owner without business", commands.AddServiceInput{OwnerID: 99, Service: domain.ServiceInput{Name: "x"}}, business.ErrBusinessNotFound},
		{"invalid service", commands.AddServiceInput{OwnerID: 7}, domain.ErrServiceNameInvalid},
		{"branch of another business", commands.AddServiceInput{OwnerID: 7, Service: domain.ServiceInput{Name: "x", BranchID: theirBranch.ID}}, business.ErrBranchNotFound},
		{"branch that does not exist", commands.AddServiceInput{OwnerID: 7, Service: domain.ServiceInput{Name: "x", BranchID: 999}}, business.ErrBranchNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := add.Handle(t.Context(), tt.input); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
	if len(services.byID) != 0 {
		t.Errorf("stored services = %d, a rejected service must not be stored", len(services.byID))
	}
}

func TestServiceLimitCountsOnlyActiveServices(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	services := newFakeServices()
	ent := fixedEntitlements{limits: limits{services: 2}}
	add := commands.NewAddService(businesses, branches, services, ent)
	setActive := commands.NewSetServiceActive(businesses, services, ent)

	first := addService(t, add, 7, "One")
	second := addService(t, add, 7, "Two")
	if _, err := add.Handle(t.Context(), commands.AddServiceInput{OwnerID: 7, Service: domain.ServiceInput{Name: "Three"}}); !errors.Is(err, business.ErrServiceLimitReached) {
		t.Fatalf("third service error = %v, want ErrServiceLimitReached", err)
	}
	addService(t, add, 8, "Other business")

	if _, err := setActive.Handle(t.Context(), commands.SetServiceActiveInput{OwnerID: 7, ServiceID: first.ID, Active: false}); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	third := addService(t, add, 7, "Three")

	// The limit is full again (second + third): activating the first must fail,
	// but deactivating or re-activating an already active one must not.
	if _, err := setActive.Handle(t.Context(), commands.SetServiceActiveInput{OwnerID: 7, ServiceID: first.ID, Active: true}); !errors.Is(err, business.ErrServiceLimitReached) {
		t.Errorf("activation over the limit error = %v, want ErrServiceLimitReached", err)
	}
	if got, err := setActive.Handle(t.Context(), commands.SetServiceActiveInput{OwnerID: 7, ServiceID: second.ID, Active: true}); err != nil || !got.Active {
		t.Errorf("activating an already active service = %+v, %v; want a no-op", got, err)
	}
	if services.byID[first.ID].Active {
		t.Error("a rejected activation must leave the service inactive")
	}
	if _, err := setActive.Handle(t.Context(), commands.SetServiceActiveInput{OwnerID: 7, ServiceID: third.ID, Active: false}); err != nil {
		t.Errorf("deactivating must always be allowed: %v", err)
	}
}

func TestEditAndRemoveServiceCannotReachAnotherBusiness(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	services := newFakeServices()
	mine := addService(t, commands.NewAddService(businesses, branches, services, unlimited()), 7, "Mine")

	edit := commands.NewEditService(businesses, branches, services)
	renamed := domain.ServiceInput{Name: "Renamed"}
	if got, err := edit.Handle(t.Context(), commands.EditServiceInput{OwnerID: 7, ServiceID: mine.ID, Service: renamed}); err != nil || got.Name != "Renamed" {
		t.Fatalf("own edit = %+v, %v", got, err)
	}
	renamed.Name = "Hijacked"
	if _, err := edit.Handle(t.Context(), commands.EditServiceInput{OwnerID: 8, ServiceID: mine.ID, Service: renamed}); !errors.Is(err, business.ErrServiceNotFound) {
		t.Fatalf("foreign edit error = %v, want ErrServiceNotFound", err)
	}
	if services.byID[mine.ID].Name != "Renamed" {
		t.Errorf("name = %q, a foreign edit must not change it", services.byID[mine.ID].Name)
	}

	remove := commands.NewRemoveService(businesses, services)
	if err := remove.Handle(t.Context(), commands.RemoveServiceInput{OwnerID: 8, ServiceID: mine.ID}); !errors.Is(err, business.ErrServiceNotFound) {
		t.Fatalf("foreign remove error = %v, want ErrServiceNotFound", err)
	}
	if err := remove.Handle(t.Context(), commands.RemoveServiceInput{OwnerID: 7, ServiceID: mine.ID}); err != nil {
		t.Fatalf("own remove: %v", err)
	}
	if len(services.byID) != 0 {
		t.Error("the service must be gone")
	}
}

func TestEditServiceKeepsItsActiveFlagAndChecksTheBranch(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	services := newFakeServices()
	mine := addService(t, commands.NewAddService(businesses, branches, services, unlimited()), 7, "Mine")
	if _, err := commands.NewSetServiceActive(businesses, services, unlimited()).Handle(t.Context(), commands.SetServiceActiveInput{OwnerID: 7, ServiceID: mine.ID, Active: false}); err != nil {
		t.Fatalf("deactivate: %v", err)
	}

	edit := commands.NewEditService(businesses, branches, services)
	got, err := edit.Handle(t.Context(), commands.EditServiceInput{OwnerID: 7, ServiceID: mine.ID, Service: domain.ServiceInput{Name: "Edited"}})
	if err != nil || got.Active {
		t.Fatalf("edit = %+v, %v; want it to stay inactive", got, err)
	}
	if _, err := edit.Handle(t.Context(), commands.EditServiceInput{OwnerID: 7, ServiceID: mine.ID, Service: domain.ServiceInput{Name: "x", BranchID: 999}}); !errors.Is(err, business.ErrBranchNotFound) {
		t.Errorf("unknown branch error = %v, want ErrBranchNotFound", err)
	}
}

func TestSetServiceSchedule(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	services := newFakeServices()
	mine := addService(t, commands.NewAddService(businesses, branches, services, unlimited()), 7, "Mine")
	set := commands.NewSetServiceSchedule(businesses, services)

	week := []domain.Slot{{Weekday: 1, Start: 9 * 60, End: 10 * 60}, {Weekday: 1, Start: 18 * 60, End: 19 * 60}}
	if err := set.Handle(t.Context(), commands.SetServiceScheduleInput{OwnerID: 7, ServiceID: mine.ID, Slots: week}); err != nil {
		t.Fatalf("set schedule: %v", err)
	}

	overlapping := []domain.Slot{{Weekday: 1, Start: 9 * 60, End: 11 * 60}, {Weekday: 1, Start: 10 * 60, End: 12 * 60}}
	if err := set.Handle(t.Context(), commands.SetServiceScheduleInput{OwnerID: 7, ServiceID: mine.ID, Slots: overlapping}); !errors.Is(err, domain.ErrSlotsOverlap) {
		t.Errorf("overlap error = %v, want ErrSlotsOverlap", err)
	}
	if len(services.schedules[mine.ID]) != 2 {
		t.Error("a rejected schedule must leave the previous one untouched")
	}
	if err := set.Handle(t.Context(), commands.SetServiceScheduleInput{OwnerID: 8, ServiceID: mine.ID, Slots: week}); !errors.Is(err, business.ErrServiceNotFound) {
		t.Errorf("foreign schedule error = %v, want ErrServiceNotFound", err)
	}
	if err := set.Handle(t.Context(), commands.SetServiceScheduleInput{OwnerID: 7, ServiceID: mine.ID}); err != nil || len(services.schedules[mine.ID]) != 0 {
		t.Errorf("clearing the schedule = %v, left %d slots", err, len(services.schedules[mine.ID]))
	}
}
