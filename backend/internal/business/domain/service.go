package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Validation errors for services.
var (
	ErrServiceNameInvalid        = errors.New("business: service name is not valid")
	ErrServiceDescriptionInvalid = errors.New("business: service description is not valid")
	// ErrBranchNotFoundForService means the branch id cannot refer to any branch.
	ErrBranchNotFoundForService = errors.New("business: service branch is not valid")
)

const (
	maxServiceNameLen        = 100
	maxServiceDescriptionLen = 500
)

// Service is something a business offers, such as a yoga class or a court rental.
// The weekly schedule is a list of Slot kept apart from the service.
type Service struct {
	ID          int64
	BusinessID  int64
	BranchID    int64 // 0 when the service is offered by the whole business
	Name        string
	Description string
	Active      bool
}

// ServiceInput is what an owner provides to create or edit a service.
type ServiceInput struct {
	Name        string
	Description string
	BranchID    int64
}

// NewService validates the input and builds an active service. BusinessID and ID
// are assigned by the use case and the repository.
func NewService(in ServiceInput) (Service, error) {
	return Service{Active: true}.Update(in)
}

// Update validates the input and returns the service with the new details,
// keeping its identity and whether it is active.
func (s Service) Update(in ServiceInput) (Service, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > maxServiceNameLen {
		return Service{}, ErrServiceNameInvalid
	}
	description := strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(description) > maxServiceDescriptionLen {
		return Service{}, ErrServiceDescriptionInvalid
	}
	if in.BranchID < 0 {
		return Service{}, ErrBranchNotFoundForService
	}

	s.Name, s.Description, s.BranchID = name, description, in.BranchID
	return s, nil
}

// Activate turns the service on. Whether the plan has room is the use case's call.
func (s Service) Activate() Service {
	s.Active = true
	return s
}

// Deactivate turns the service off without deleting it.
func (s Service) Deactivate() Service {
	s.Active = false
	return s
}
