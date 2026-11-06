package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

const validPassword = "correct horse battery"

func TestNormalizeEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr error
	}{
		{"already normal", "owner@example.com", "owner@example.com", nil},
		{"uppercase and padding", "  Owner@Example.COM \n", "owner@example.com", nil},
		{"plus tag is kept", "owner+gym@example.com", "owner+gym@example.com", nil},
		{"empty", "", "", domain.ErrEmailInvalid},
		{"blank", "   ", "", domain.ErrEmailInvalid},
		{"no at sign", "owner.example.com", "", domain.ErrEmailInvalid},
		{"no domain", "owner@", "", domain.ErrEmailInvalid},
		{"domain without dot", "owner@localhost", "", domain.ErrEmailInvalid},
		{"display name", "Owner <owner@example.com>", "", domain.ErrEmailInvalid},
		{"two addresses", "a@example.com, b@example.com", "", domain.ErrEmailInvalid},
		{"too long", strings.Repeat("a", 250) + "@example.com", "", domain.ErrEmailInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.NormalizeEmail(tt.raw)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NormalizeEmail(%q) error = %v, want %v", tt.raw, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("NormalizeEmail(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseRole(t *testing.T) {
	t.Parallel()

	for raw, want := range map[string]domain.Role{
		"business_admin": domain.RoleBusinessAdmin,
		"super_admin":    domain.RoleSuperAdmin,
	} {
		got, err := domain.ParseRole(raw)
		if err != nil || got != want {
			t.Errorf("ParseRole(%q) = (%q, %v), want (%q, nil)", raw, got, err, want)
		}
	}

	for _, raw := range []string{"", "customer", "Super_Admin", "admin"} {
		if _, err := domain.ParseRole(raw); !errors.Is(err, domain.ErrRoleInvalid) {
			t.Errorf("ParseRole(%q) error = %v, want %v", raw, err, domain.ErrRoleInvalid)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{"minimum length", strings.Repeat("a", 10), nil},
		{"maximum length", strings.Repeat("a", 128), nil},
		{"one below minimum", strings.Repeat("a", 9), domain.ErrPasswordTooShort},
		{"empty", "", domain.ErrPasswordTooShort},
		{"one above maximum", strings.Repeat("a", 129), domain.ErrPasswordTooLong},
		{"multibyte counts characters", strings.Repeat("ñ", 10), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := domain.ValidatePassword(tt.password); !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidatePassword(len %d) error = %v, want %v", len(tt.password), err, tt.wantErr)
			}
		})
	}
}

func TestNewUserNormalizesAndHashes(t *testing.T) {
	t.Parallel()

	user, err := domain.NewUser("  Owner@Example.com ", validPassword, domain.RoleBusinessAdmin)
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}

	if user.Email != "owner@example.com" {
		t.Errorf("Email = %q, want it normalized", user.Email)
	}
	if user.Role != domain.RoleBusinessAdmin {
		t.Errorf("Role = %q, want %q", user.Role, domain.RoleBusinessAdmin)
	}
	if strings.Contains(user.PasswordHash, validPassword) {
		t.Error("PasswordHash contains the plain password")
	}
	if !user.CheckPassword(validPassword) {
		t.Error("CheckPassword(correct) = false, want true")
	}
	if user.CheckPassword("wrong password!") {
		t.Error("CheckPassword(wrong) = true, want false")
	}
}

func TestNewUserRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		email    string
		password string
		role     domain.Role
		wantErr  error
	}{
		{"bad email", "nope", validPassword, domain.RoleBusinessAdmin, domain.ErrEmailInvalid},
		{"short password", "owner@example.com", "short", domain.RoleBusinessAdmin, domain.ErrPasswordTooShort},
		{"unknown role", "owner@example.com", validPassword, domain.Role("customer"), domain.ErrRoleInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.NewUser(tt.email, tt.password, tt.role); !errors.Is(err, tt.wantErr) {
				t.Errorf("NewUser error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckPasswordIsFalseForAnUnreadableHash(t *testing.T) {
	t.Parallel()

	user := domain.User{PasswordHash: "not-a-phc-string"}
	if user.CheckPassword(validPassword) {
		t.Error("CheckPassword with a malformed hash = true, want false")
	}
}
