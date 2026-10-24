package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func TestNewService(t *testing.T) {
	t.Parallel()

	got, err := domain.NewService(domain.ServiceInput{Name: "  Yoga Flow ", Description: " Para todos ", BranchID: 4})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if got.Name != "Yoga Flow" || got.Description != "Para todos" || got.BranchID != 4 {
		t.Errorf("service = %+v, want trimmed values and the branch", got)
	}
	if !got.Active {
		t.Error("a new service must be active")
	}

	if got, err := domain.NewService(domain.ServiceInput{Name: "Cancha"}); err != nil || got.BranchID != 0 {
		t.Errorf("a service without branch = %+v, %v; want it allowed", got, err)
	}

	tests := []struct {
		name  string
		input domain.ServiceInput
		want  error
	}{
		{"empty name", domain.ServiceInput{Name: " "}, domain.ErrServiceNameInvalid},
		{"name too long", domain.ServiceInput{Name: strings.Repeat("a", 101)}, domain.ErrServiceNameInvalid},
		{"description too long", domain.ServiceInput{Name: "x", Description: strings.Repeat("a", 501)}, domain.ErrServiceDescriptionInvalid},
		{"negative branch", domain.ServiceInput{Name: "x", BranchID: -1}, domain.ErrBranchNotFoundForService},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.NewService(tt.input); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestServiceUpdateKeepsIdentityAndState(t *testing.T) {
	t.Parallel()

	current := domain.Service{ID: 3, BusinessID: 2, Active: false}
	got, err := current.Update(domain.ServiceInput{Name: "Spinning", BranchID: 9})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ID != 3 || got.BusinessID != 2 || got.Active {
		t.Errorf("identity/state = (%d, %d, %v), want (3, 2, false)", got.ID, got.BusinessID, got.Active)
	}
	if got.Name != "Spinning" || got.BranchID != 9 {
		t.Errorf("service = %+v", got)
	}
	if _, err := current.Update(domain.ServiceInput{}); !errors.Is(err, domain.ErrServiceNameInvalid) {
		t.Errorf("invalid update error = %v, want ErrServiceNameInvalid", err)
	}
}

func TestServiceActivation(t *testing.T) {
	t.Parallel()

	s := domain.Service{Active: true}
	if s.Deactivate().Active {
		t.Error("Deactivate must turn the service off")
	}
	if !s.Deactivate().Activate().Active {
		t.Error("Activate must turn the service on")
	}
}
