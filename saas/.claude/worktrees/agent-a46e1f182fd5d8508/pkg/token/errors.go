package token

import "errors"

// Sentinel errors surfaced by token parsing and verification.
//
// All sentinels are wrapped (via fmt.Errorf with %w) where they originate, so
// callers should match with errors.Is rather than equality. Every message
// begins with the "token: " prefix to keep error chains self-identifying.
var (
	// ErrInvalid is the catch-all for malformed or otherwise-invalid tokens
	// (missing kid header, malformed JWT, nbf in the future, …). It is the
	// broadest bucket; more specific causes use the sentinels below.
	ErrInvalid = errors.New("token: invalid token")

	// ErrExpired indicates the token is past its exp claim, accounting for
	// any configured clock skew leeway.
	ErrExpired = errors.New("token: expired")

	// ErrWrongAudience indicates none of the configured audiences appear in
	// the token's aud claim.
	ErrWrongAudience = errors.New("token: wrong audience")

	// ErrWrongIssuer indicates the token's iss claim does not match the
	// configured issuer.
	ErrWrongIssuer = errors.New("token: wrong issuer")

	// ErrUnknownKID indicates the verifier has no public key registered for
	// the kid carried in the token header.
	ErrUnknownKID = errors.New("token: unknown kid")

	// ErrSignature indicates the token's signature failed verification or
	// its alg header does not match the verifier's expectation.
	ErrSignature = errors.New("token: invalid signature")
)
