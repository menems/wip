package token

import (
	"errors"
	"fmt"
	"time"
)

// Config bundles the parameters required to construct a [Manager]. All
// required fields are validated by [NewManager]; zero or missing values cause
// construction to fail rather than producing a half-built manager.
//
// Plain-struct configuration is preferred over functional options here
// because every required dependency (issuer, audience, key provider, …) is
// load-bearing and must be supplied together. This matches the style of
// other pkg/* configuration objects in this repository.
type Config struct {
	// Issuer is the value written to and required from the JWT "iss" claim.
	// Must be non-empty.
	Issuer string

	// Audience is the list of acceptable values for the JWT "aud" claim.
	// At least one entry is required. A token is accepted when its "aud"
	// claim overlaps with this list by at least one value.
	Audience []string

	// TTL is the lifetime applied to newly issued tokens via the "exp"
	// claim. Must be strictly positive.
	TTL time.Duration

	// Skew is the leeway tolerated during "exp" and "nbf" validation, and is
	// also subtracted from "nbf" at issue time. Zero disables both. Must be
	// non-negative.
	Skew time.Duration

	// KeyProvider resolves the active signing key (via [KeyProvider.SigningKey])
	// and the kid-indexed verification keys (via [KeyProvider.VerificationKey]).
	// Required.
	KeyProvider KeyProvider

	// Signers maps each supported JWT "alg" header value (for example "ES256"
	// or "RS256") to the [Signer] capable of producing and validating tokens
	// for that algorithm. Manager picks the signer that matches the active
	// signing key at issue time, and the signer that matches the token's
	// "alg" header at parse time. At least one entry is required.
	Signers map[string]Signer

	// Clock optionally overrides time.Now. Useful for deterministic tests.
	Clock func() time.Time
}

// validate ensures every required field is set and self-consistent. It does
// not query KeyProvider — that is deferred until the first Issue/Parse so a
// transient lookup error never blocks construction.
func (c Config) validate() error {
	if c.Issuer == "" {
		return errors.New("token: Config.Issuer is required")
	}
	if len(c.Audience) == 0 {
		return errors.New("token: Config.Audience must contain at least one entry")
	}
	for i, aud := range c.Audience {
		if aud == "" {
			return fmt.Errorf("token: Config.Audience[%d] is empty", i)
		}
	}
	if c.TTL <= 0 {
		return fmt.Errorf("token: Config.TTL must be positive, got %s", c.TTL)
	}
	if c.Skew < 0 {
		return fmt.Errorf("token: Config.Skew must be non-negative, got %s", c.Skew)
	}
	if c.KeyProvider == nil {
		return errors.New("token: Config.KeyProvider is required")
	}
	if len(c.Signers) == 0 {
		return errors.New("token: Config.Signers must contain at least one entry")
	}
	for alg, s := range c.Signers {
		if alg == "" {
			return errors.New("token: Config.Signers has an empty alg key")
		}
		if s == nil {
			return fmt.Errorf("token: Config.Signers[%q] is nil", alg)
		}
		if s.Alg() != alg {
			return fmt.Errorf("token: Config.Signers[%q] reports alg %q", alg, s.Alg())
		}
	}
	return nil
}
