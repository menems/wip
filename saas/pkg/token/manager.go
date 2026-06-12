package token

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RegisteredClaimsSetter is the optional contract that an application claims
// type implements so that [Manager.Issue] can populate the standard
// registered fields (iss/sub/aud/iat/nbf/exp/jti) on the caller's behalf.
//
// A typical implementation embeds [jwt.RegisteredClaims] and forwards the
// setter to the embedded value:
//
//	type AppClaims struct {
//	    jwt.RegisteredClaims
//	    UserID string `json:"uid"`
//	}
//
//	func (c *AppClaims) SetRegisteredClaims(rc jwt.RegisteredClaims) {
//	    c.RegisteredClaims = rc
//	}
//
// Implementing this interface is required: [Manager.Issue] refuses to sign
// claims that cannot be augmented with the issuer/audience/timing data the
// rest of the package depends on.
type RegisteredClaimsSetter interface {
	SetRegisteredClaims(jwt.RegisteredClaims)
}

// Manager issues and verifies JWTs of a specific application claims type T.
//
// Manager is safe for concurrent use: it holds only immutable configuration
// after construction. The underlying [Signer] and [KeyProvider] instances
// must themselves be concurrency-safe (the implementations in this package
// already are).
type Manager[T Claims] struct {
	issuer      string
	audience    []string
	ttl         time.Duration
	skew        time.Duration
	keys        KeyProvider
	signers     map[string]Signer
	clock       func() time.Time
	newClaims   func() T
}

// NewManager validates cfg and returns a ready-to-use [Manager]. The factory
// callback newClaimsT is invoked by [Manager.Parse] to allocate an empty T
// into which the parsed claims are decoded; it must return a value whose
// underlying claims fields are addressable (typically a pointer to a struct
// that embeds [jwt.RegisteredClaims]).
//
// Returning the empty value via a caller-supplied factory keeps Manager free
// of reflection while still letting Parse hand back a typed T.
func NewManager[T Claims](cfg Config, newClaimsT func() T) (*Manager[T], error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if newClaimsT == nil {
		return nil, fmt.Errorf("token: NewManager requires a non-nil claims factory")
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	return &Manager[T]{
		issuer:    cfg.Issuer,
		audience:  append([]string(nil), cfg.Audience...),
		ttl:       cfg.TTL,
		skew:      cfg.Skew,
		keys:      cfg.KeyProvider,
		signers:   cloneSigners(cfg.Signers),
		clock:     clock,
		newClaims: newClaimsT,
	}, nil
}

// Issue populates the registered claims on the supplied T (iss/aud/iat/nbf/
// exp/jti, with sub preserved from the caller) and returns a compact signed
// JWT. The active signing key from the configured KeyProvider determines
// both the kid header and which signer is used.
//
// T must implement [RegisteredClaimsSetter]; the call fails otherwise so
// that nothing slips through unsigned by the issuer/audience contract.
func (m *Manager[T]) Issue(ctx context.Context, claims T) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	setter, ok := any(claims).(RegisteredClaimsSetter)
	if !ok {
		return "", fmt.Errorf("token: claims type %T must implement RegisteredClaimsSetter", claims)
	}

	kid, signingKey, err := m.keys.SigningKey()
	if err != nil {
		return "", fmt.Errorf("token: load signing key: %w", err)
	}
	if kid == "" {
		return "", fmt.Errorf("token: signing key has empty kid")
	}
	alg, err := algForSigner(signingKey)
	if err != nil {
		return "", err
	}
	signer, ok := m.signers[alg]
	if !ok {
		return "", fmt.Errorf("token: no signer registered for alg %q", alg)
	}

	now := m.clock()
	subject := preserveSubject(claims)
	rc := NewRegisteredClaims(now, IssueParams{
		Issuer:   m.issuer,
		Subject:  subject,
		Audience: m.audience,
		TTL:      m.ttl,
		Skew:     m.skew,
	})
	setter.SetRegisteredClaims(rc)

	raw, err := signer.Sign(claims, kid)
	if err != nil {
		return "", fmt.Errorf("token: sign: %w", err)
	}
	return raw, nil
}

// Parse verifies raw against the configured KeyProvider/Signers and the
// issuer/audience/timing rules in Config. On success it returns a freshly
// allocated T populated from the token's payload.
//
// Validation order: kid header → verification key lookup → alg vs key-type
// agreement → signature → iss → aud → exp → nbf. Every failure produces a
// descriptive error message; typed sentinels arrive in a later step.
func (m *Manager[T]) Parse(ctx context.Context, raw string) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}

	// Step 1: peek at the header without verification to find kid and alg.
	parsed, _, err := jwt.NewParser().ParseUnverified(raw, jwt.MapClaims{})
	if err != nil {
		return zero, fmt.Errorf("token: parse header: %w", err)
	}
	kid, _ := parsed.Header["kid"].(string)
	if kid == "" {
		return zero, fmt.Errorf("token: missing kid header")
	}
	alg, _ := parsed.Header["alg"].(string)
	if alg == "" {
		return zero, fmt.Errorf("token: missing alg header")
	}

	// Step 2: resolve the verification key for that kid.
	pub, err := m.keys.VerificationKey(kid)
	if err != nil {
		return zero, fmt.Errorf("token: verification key for kid %q: %w", kid, err)
	}

	// Step 3: the alg header must match the key type.
	keyAlg, err := algForPublicKey(pub)
	if err != nil {
		return zero, err
	}
	if keyAlg != alg {
		return zero, fmt.Errorf("token: alg %q does not match key type for kid %q (expected %q)", alg, kid, keyAlg)
	}

	// Step 4: signature verification. We bypass [Signer.Verify] here because
	// the underlying jwt/v5 parser also runs its own time-based claim
	// validation, which would conflict with the deterministic clock used by
	// Manager (and would, for example, mark every test-clock token as
	// expired). We restrict the accepted alg to the configured signer's, but
	// disable claim validation and run our own below.
	if _, ok := m.signers[alg]; !ok {
		return zero, fmt.Errorf("token: no signer registered for alg %q", alg)
	}
	out := m.newClaims()
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{alg}),
		jwt.WithoutClaimsValidation(),
	)
	if _, err := parser.ParseWithClaims(raw, out, func(t *jwt.Token) (any, error) {
		return pub, nil
	}); err != nil {
		return zero, fmt.Errorf("token: verify signature: %w", err)
	}

	// Steps 5-8: registered-claims invariants.
	if err := m.validateRegistered(out); err != nil {
		return zero, err
	}
	return out, nil
}

func (m *Manager[T]) validateRegistered(c T) error {
	issuer, err := c.GetIssuer()
	if err != nil {
		return fmt.Errorf("token: read issuer: %w", err)
	}
	if issuer != m.issuer {
		return fmt.Errorf("token: issuer %q does not match expected %q", issuer, m.issuer)
	}

	aud, err := c.GetAudience()
	if err != nil {
		return fmt.Errorf("token: read audience: %w", err)
	}
	if !audienceOverlaps(aud, m.audience) {
		return fmt.Errorf("token: audience %v does not overlap configured %v", []string(aud), m.audience)
	}

	now := m.clock()

	exp, err := c.GetExpirationTime()
	if err != nil {
		return fmt.Errorf("token: read exp: %w", err)
	}
	if exp == nil {
		return fmt.Errorf("token: missing exp claim")
	}
	if now.After(exp.Time.Add(m.skew)) {
		return fmt.Errorf("token: expired at %s (now %s, skew %s)", exp.Time.UTC(), now.UTC(), m.skew)
	}

	nbf, err := c.GetNotBefore()
	if err != nil {
		return fmt.Errorf("token: read nbf: %w", err)
	}
	if nbf != nil && now.Add(m.skew).Before(nbf.Time) {
		return fmt.Errorf("token: not yet valid: nbf %s (now %s, skew %s)", nbf.Time.UTC(), now.UTC(), m.skew)
	}
	return nil
}

// preserveSubject extracts the caller-supplied "sub" so that Manager can
// re-emit it via NewRegisteredClaims without depending on T's concrete type.
// Errors are swallowed: a missing or unreadable subject simply produces an
// empty string, which NewRegisteredClaims encodes as "no subject".
func preserveSubject(c jwt.Claims) string {
	sub, err := c.GetSubject()
	if err != nil {
		return ""
	}
	return sub
}

// algForSigner maps a concrete crypto.Signer to the JWT alg header value it
// produces. Only the algorithms supported by this package are recognised.
func algForSigner(s any) (string, error) {
	switch s.(type) {
	case *ecdsa.PrivateKey:
		return jwt.SigningMethodES256.Alg(), nil
	case *rsa.PrivateKey:
		return jwt.SigningMethodRS256.Alg(), nil
	default:
		return "", fmt.Errorf("token: unsupported signing key type %T", s)
	}
}

// algForPublicKey is the counterpart of [algForSigner] for verification keys.
func algForPublicKey(k any) (string, error) {
	switch k.(type) {
	case *ecdsa.PublicKey:
		return jwt.SigningMethodES256.Alg(), nil
	case *rsa.PublicKey:
		return jwt.SigningMethodRS256.Alg(), nil
	default:
		return "", fmt.Errorf("token: unsupported verification key type %T", k)
	}
}

func audienceOverlaps(token jwt.ClaimStrings, accepted []string) bool {
	for _, a := range accepted {
		for _, t := range token {
			if a == t {
				return true
			}
		}
	}
	return false
}

func cloneSigners(in map[string]Signer) map[string]Signer {
	out := make(map[string]Signer, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
