package crypto_test

import (
	"testing"

	"github.com/menems/saas/pkg/crypto"
)

func TestHashPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "securepassword", false},
		{"empty password", "", false}, // bcrypt accepts empty input
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hash, err := crypto.HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HashPassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
			if !tt.wantErr && hash == "" {
				t.Error("HashPassword() returned empty hash")
			}
		})
	}
}

func TestComparePassword(t *testing.T) {
	t.Parallel()

	hash, err := crypto.HashPassword("correctpassword")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	tests := []struct {
		name     string
		hash     string
		password string
		wantErr  bool
	}{
		{"correct password", hash, "correctpassword", false},
		{"wrong password", hash, "wrongpassword", true},
		{"empty password against valid hash", hash, "", true},
		{"empty hash", "", "anypassword", true},
		{"invalid hash", "notahash", "anypassword", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := crypto.ComparePassword(tt.hash, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ComparePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
