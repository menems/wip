package token_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/menems/saas/pkg/token"
)

// appClaims is a representative application claims struct. The static
// assertion below proves it satisfies the [token.Claims] constraint, which
// is what the upcoming generic Manager will accept.
type appClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
}

var _ token.Claims = (*appClaims)(nil)

func TestNewRegisteredClaims_PopulatesAllFields(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	params := token.IssueParams{
		Issuer:   "issuer-x",
		Subject:  "user-42",
		Audience: []string{"aud-a", "aud-b"},
		TTL:      15 * time.Minute,
		Skew:     30 * time.Second,
		ID:       "tok-123",
	}

	rc := token.NewRegisteredClaims(now, params)

	if rc.Issuer != params.Issuer {
		t.Errorf("Issuer = %q, want %q", rc.Issuer, params.Issuer)
	}
	if rc.Subject != params.Subject {
		t.Errorf("Subject = %q, want %q", rc.Subject, params.Subject)
	}
	if got := []string(rc.Audience); !equalStrings(got, params.Audience) {
		t.Errorf("Audience = %v, want %v", got, params.Audience)
	}
	if rc.ID != params.ID {
		t.Errorf("ID = %q, want %q", rc.ID, params.ID)
	}
	if rc.IssuedAt == nil || !rc.IssuedAt.Equal(now) {
		t.Errorf("IssuedAt = %v, want %v", rc.IssuedAt, now)
	}
	wantNBF := now.Add(-params.Skew)
	if rc.NotBefore == nil || !rc.NotBefore.Equal(wantNBF) {
		t.Errorf("NotBefore = %v, want %v", rc.NotBefore, wantNBF)
	}
	wantExp := now.Add(params.TTL)
	if rc.ExpiresAt == nil || !rc.ExpiresAt.Equal(wantExp) {
		t.Errorf("ExpiresAt = %v, want %v", rc.ExpiresAt, wantExp)
	}
}

func TestNewRegisteredClaims_GeneratesIDWhenEmpty(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	rc1 := token.NewRegisteredClaims(now, token.IssueParams{TTL: time.Minute})
	rc2 := token.NewRegisteredClaims(now, token.IssueParams{TTL: time.Minute})

	if rc1.ID == "" {
		t.Fatal("ID was not auto-populated")
	}
	if _, err := uuid.Parse(rc1.ID); err != nil {
		t.Errorf("auto ID %q is not a valid UUID: %v", rc1.ID, err)
	}
	if rc1.ID == rc2.ID {
		t.Errorf("two auto-generated IDs are identical: %q", rc1.ID)
	}
}

func TestNewRegisteredClaims_NoAudienceWhenEmpty(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	rc := token.NewRegisteredClaims(now, token.IssueParams{TTL: time.Minute})

	if len(rc.Audience) != 0 {
		t.Errorf("Audience = %v, want empty", rc.Audience)
	}
}

func TestNewRegisteredClaims_AudienceIsCopied(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	src := []string{"aud-a"}
	rc := token.NewRegisteredClaims(now, token.IssueParams{
		TTL:      time.Minute,
		Audience: src,
	})

	src[0] = "mutated"
	if rc.Audience[0] != "aud-a" {
		t.Errorf("Audience tracked caller slice: got %q", rc.Audience[0])
	}
}

func TestNewRegisteredClaims_ZeroSkewLeavesNBFEqualToIAT(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	rc := token.NewRegisteredClaims(now, token.IssueParams{TTL: time.Minute})

	if rc.NotBefore == nil || !rc.NotBefore.Equal(now) {
		t.Errorf("NotBefore = %v, want %v", rc.NotBefore, now)
	}
	if rc.IssuedAt == nil || !rc.IssuedAt.Equal(now) {
		t.Errorf("IssuedAt = %v, want %v", rc.IssuedAt, now)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
