package user_test

import (
	"testing"

	"github.com/menems/saas/internal/user"
)

func TestValidateEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "alice@example.com", false},
		{"valid email with subdomain", "bob@mail.example.com", false},
		{"empty email", "", true},
		{"missing at sign", "aliceexample.com", true},
		{"missing domain", "alice@", true},
		{"missing local part", "@example.com", true},
		{"spaces in email", "alice @example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := user.ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "securepassword", false},
		{"exactly 8 chars", "12345678", false},
		{"too short", "short", true},
		{"empty password", "", true},
		{"7 chars", "1234567", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := user.ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		role    user.Role
		wantErr bool
	}{
		{"admin role", user.RoleAdmin, false},
		{"member role", user.RoleMember, false},
		{"unknown role", user.Role("superuser"), true},
		{"empty role", user.Role(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := user.ValidateRole(tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRole(%q) error = %v, wantErr %v", tt.role, err, tt.wantErr)
			}
		})
	}
}

func TestUser_Validate(t *testing.T) {
	t.Parallel()

	validUser := func() user.User {
		return user.User{
			Email: "alice@example.com",
			Name:  "Alice",
			Role:  user.RoleAdmin,
		}
	}

	tests := []struct {
		name    string
		modify  func(*user.User)
		wantErr bool
	}{
		{"valid user", func(u *user.User) {}, false},
		{"empty name", func(u *user.User) { u.Name = "" }, true},
		{"invalid email", func(u *user.User) { u.Email = "bad" }, true},
		{"invalid role", func(u *user.User) { u.Role = "unknown" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := validUser()
			tt.modify(&u)
			err := u.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("User.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
