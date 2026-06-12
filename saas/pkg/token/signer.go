package token

// Signer is the low-level primitive that owns the algorithm wiring on top of
// github.com/golang-jwt/jwt/v5. It signs claims with a specific algorithm and
// verifies that a raw token was produced by the matching algorithm and key.
// Full claims validation (iss/aud/exp/nbf) is the responsibility of the
// higher-level Manager (see step-05 of the pkg-token plan).

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// minRSABits is the smallest RSA modulus size accepted for RS256 signing.
const minRSABits = 2048

// Signer signs and verifies JWTs with a single asymmetric algorithm.
//
// Verify only checks the signature, algorithm, and (when set) the kid header;
// it does not enforce time-based or audience claims. Callers that need full
// validation should compose Signer with the Manager in step-05.
type Signer interface {
	// Alg returns the JWT "alg" header value used by this signer
	// (for example "ES256" or "RS256").
	Alg() string
	// Sign serialises claims into a signed compact JWT. If kid is non-empty
	// it is written to the "kid" header.
	Sign(claims jwt.Claims, kid string) (string, error)
	// Verify parses raw, enforces that it was signed with Alg(), and
	// populates claims with the decoded payload. The supplied claims value
	// must be a pointer to a jwt.Claims-compatible struct, exactly as
	// required by jwt.ParseWithClaims.
	Verify(raw string, claims jwt.Claims) error
}

// NewES256Signer constructs a Signer that signs with ECDSA P-256 (ES256) and
// verifies against the provided public key. Both keys must use the P-256 curve.
func NewES256Signer(priv *ecdsa.PrivateKey, pub *ecdsa.PublicKey) (Signer, error) {
	if priv == nil {
		return nil, errors.New("token: ES256 private key is nil")
	}
	if pub == nil {
		return nil, errors.New("token: ES256 public key is nil")
	}
	if priv.Curve != elliptic.P256() {
		return nil, fmt.Errorf("token: ES256 private key must use P-256, got %s", priv.Curve.Params().Name)
	}
	if pub.Curve != elliptic.P256() {
		return nil, fmt.Errorf("token: ES256 public key must use P-256, got %s", pub.Curve.Params().Name)
	}
	return &es256Signer{priv: priv, pub: pub}, nil
}

// NewRS256Signer constructs a Signer that signs with RSA-SHA256 (RS256). The
// private key (and public key) must have a modulus of at least 2048 bits.
func NewRS256Signer(priv *rsa.PrivateKey, pub *rsa.PublicKey) (Signer, error) {
	if priv == nil {
		return nil, errors.New("token: RS256 private key is nil")
	}
	if pub == nil {
		return nil, errors.New("token: RS256 public key is nil")
	}
	if bits := priv.N.BitLen(); bits < minRSABits {
		return nil, fmt.Errorf("token: RS256 private key must be at least %d bits, got %d", minRSABits, bits)
	}
	if bits := pub.N.BitLen(); bits < minRSABits {
		return nil, fmt.Errorf("token: RS256 public key must be at least %d bits, got %d", minRSABits, bits)
	}
	return &rs256Signer{priv: priv, pub: pub}, nil
}

type es256Signer struct {
	priv *ecdsa.PrivateKey
	pub  *ecdsa.PublicKey
}

func (s *es256Signer) Alg() string { return jwt.SigningMethodES256.Alg() }

func (s *es256Signer) Sign(claims jwt.Claims, kid string) (string, error) {
	return signWith(jwt.SigningMethodES256, claims, kid, s.priv)
}

func (s *es256Signer) Verify(raw string, claims jwt.Claims) error {
	return verifyWith(raw, claims, jwt.SigningMethodES256, s.pub)
}

type rs256Signer struct {
	priv *rsa.PrivateKey
	pub  *rsa.PublicKey
}

func (s *rs256Signer) Alg() string { return jwt.SigningMethodRS256.Alg() }

func (s *rs256Signer) Sign(claims jwt.Claims, kid string) (string, error) {
	return signWith(jwt.SigningMethodRS256, claims, kid, s.priv)
}

func (s *rs256Signer) Verify(raw string, claims jwt.Claims) error {
	return verifyWith(raw, claims, jwt.SigningMethodRS256, s.pub)
}

// signWith creates a signed compact JWT using method, embedding kid in the
// header when non-empty.
func signWith(method jwt.SigningMethod, claims jwt.Claims, kid string, key any) (string, error) {
	tok := jwt.NewWithClaims(method, claims)
	if kid != "" {
		tok.Header["kid"] = kid
	}
	signed, err := tok.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("token: sign %s: %w", method.Alg(), err)
	}
	return signed, nil
}

// verifyWith parses raw, asserts the signing method matches expected, and
// populates claims on success. Algorithm and signature mismatches are reported
// as [ErrSignature]; other parse failures (malformed token, etc.) are also
// surfaced through it since Verify is purely a crypto-level check.
func verifyWith(raw string, claims jwt.Claims, expected jwt.SigningMethod, key any) error {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{expected.Alg()}))
	_, err := parser.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != expected.Alg() {
			return nil, fmt.Errorf("%w: unexpected alg %q", ErrSignature, t.Method.Alg())
		}
		return key, nil
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSignature, err)
	}
	return nil
}
