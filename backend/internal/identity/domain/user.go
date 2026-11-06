// Package domain holds the identity rules: who can have an account, with which
// role, and how passwords are stored. It imports no infrastructure.
package domain

import (
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

// Validation errors. The HTTP layer maps each one to a stable error code.
var (
	ErrEmailInvalid     = errors.New("identity: email is not valid")
	ErrRoleInvalid      = errors.New("identity: role is not valid")
	ErrPasswordTooShort = errors.New("identity: password is too short")
	ErrPasswordTooLong  = errors.New("identity: password is too long")
)

// Password length bounds, in characters. The maximum keeps a hostile request
// from making the server hash megabytes of input.
const (
	minPasswordLen = 10
	maxPasswordLen = 128
	maxEmailLen    = 254
)

// Role is what a user is allowed to do.
type Role string

// The roles of the platform. Customers have no account, so they have no role.
const (
	RoleBusinessAdmin Role = "business_admin"
	RoleSuperAdmin    Role = "super_admin"
)

// ParseRole converts the stored or transmitted spelling into a Role.
func ParseRole(raw string) (Role, error) {
	switch role := Role(raw); role {
	case RoleBusinessAdmin, RoleSuperAdmin:
		return role, nil
	default:
		return "", ErrRoleInvalid
	}
}

// User is a business owner or a super admin.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}

// NewUser validates the input and builds a User that is ready to be stored: the
// email is normalized and the password is hashed. ID and CreatedAt are assigned
// by the repository.
func NewUser(email, password string, role Role) (User, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return User{}, err
	}

	if _, err := ParseRole(string(role)); err != nil {
		return User{}, err
	}

	if err := ValidatePassword(password); err != nil {
		return User{}, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}

	return User{Email: normalized, PasswordHash: hash, Role: role}, nil
}

// CheckPassword reports whether password is the user's. An unreadable stored
// hash counts as a mismatch: it never authenticates anyone.
func (u User) CheckPassword(password string) bool {
	ok, err := VerifyPassword(u.PasswordHash, password)
	return err == nil && ok
}

// NormalizeEmail trims and lowercases raw and checks it is a single bare address
// with a dotted domain. The result is the only spelling that is stored or
// compared.
func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > maxEmailLen {
		return "", ErrEmailInvalid
	}

	// ParseAddress also accepts "Name <a@b.c>" and comment syntax; requiring the
	// parsed address to equal the input keeps only the bare form.
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", ErrEmailInvalid
	}

	_, domainPart, _ := strings.Cut(email, "@")
	if !strings.Contains(domainPart, ".") {
		return "", ErrEmailInvalid
	}

	return email, nil
}

// ValidatePassword enforces the length bounds. Composition rules are left out on
// purpose: length is what resists guessing, and the login rate limit covers the
// rest.
func ValidatePassword(password string) error {
	switch length := utf8.RuneCountInString(password); {
	case length < minPasswordLen:
		return ErrPasswordTooShort
	case length > maxPasswordLen:
		return ErrPasswordTooLong
	default:
		return nil
	}
}
