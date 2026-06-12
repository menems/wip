package token

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the type-parameter constraint accepted by the generic Manager
// (see step-05 of the pkg-token plan). It simply re-exports jwt.Claims so
// that any value satisfying the underlying JWT contract can be passed to
// the signer and parser without conversion.
//
// Application claims are expected to embed [jwt.RegisteredClaims] in order
// to inherit the iss/sub/aud/iat/exp/nbf/jti getters that jwt/v5 uses for
// standard validation. The [NewRegisteredClaims] helper below builds that
// embedded value from a small parameter struct.
type Claims interface {
	jwt.Claims
}

// IssueParams describes the standard JWT registered claims that callers
// fill in at issue time. The TTL controls the "exp" claim; Skew shifts
// "nbf" backwards by the given duration to absorb client clock drift
// (zero disables the shift). When ID is empty a random UUID is assigned so
// that every issued token has a stable jti.
type IssueParams struct {
	Issuer   string
	Subject  string
	Audience []string
	TTL      time.Duration
	Skew     time.Duration
	ID       string
}

// NewRegisteredClaims returns a populated [jwt.RegisteredClaims] suitable
// for embedding in an application-specific claims struct. now is the
// reference instant used for iat/nbf/exp; pass time.Now() in production and
// a frozen value from tests.
//
// The function performs no validation: callers are expected to enforce
// invariants (non-empty issuer/audience, positive TTL, …) at construction
// time, typically inside Manager's config check.
func NewRegisteredClaims(now time.Time, p IssueParams) jwt.RegisteredClaims {
	id := p.ID
	if id == "" {
		id = uuid.NewString()
	}
	rc := jwt.RegisteredClaims{
		Issuer:    p.Issuer,
		Subject:   p.Subject,
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now.Add(-p.Skew)),
		ExpiresAt: jwt.NewNumericDate(now.Add(p.TTL)),
		ID:        id,
	}
	if len(p.Audience) > 0 {
		rc.Audience = jwt.ClaimStrings(append([]string(nil), p.Audience...))
	}
	return rc
}
