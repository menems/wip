package token_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/menems/saas/pkg/token"
)

func newES256Signer(t *testing.T) token.Signer {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA key: %v", err)
	}
	s, err := token.NewES256Signer(priv, &priv.PublicKey)
	if err != nil {
		t.Fatalf("NewES256Signer: %v", err)
	}
	return s
}

func newRS256Signer(t *testing.T) token.Signer {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	s, err := token.NewRS256Signer(priv, &priv.PublicKey)
	if err != nil {
		t.Fatalf("NewRS256Signer: %v", err)
	}
	return s
}

func sampleClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub": "user-123",
		"iss": "test-suite",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}
}

func TestSigner_RoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		signer func(*testing.T) token.Signer
		alg    string
	}{
		{"ES256", newES256Signer, "ES256"},
		{"RS256", newRS256Signer, "RS256"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := tc.signer(t)
			if got := s.Alg(); got != tc.alg {
				t.Fatalf("Alg() = %q, want %q", got, tc.alg)
			}

			const kid = "key-1"
			raw, err := s.Sign(sampleClaims(), kid)
			if err != nil {
				t.Fatalf("Sign: %v", err)
			}

			var out jwt.MapClaims = jwt.MapClaims{}
			if err := s.Verify(raw, &out); err != nil {
				t.Fatalf("Verify: %v", err)
			}
			if out["sub"] != "user-123" {
				t.Fatalf("sub claim = %v, want user-123", out["sub"])
			}

			// kid must be recoverable from the unverified header.
			parsed, _, err := jwt.NewParser().ParseUnverified(raw, jwt.MapClaims{})
			if err != nil {
				t.Fatalf("ParseUnverified: %v", err)
			}
			if got, _ := parsed.Header["kid"].(string); got != kid {
				t.Fatalf("kid header = %q, want %q", got, kid)
			}
			if got, _ := parsed.Header["alg"].(string); got != tc.alg {
				t.Fatalf("alg header = %q, want %q", got, tc.alg)
			}
		})
	}
}

func TestSigner_TamperedTokenRejected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		signer func(*testing.T) token.Signer
	}{
		{"ES256", newES256Signer},
		{"RS256", newRS256Signer},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := tc.signer(t)
			raw, err := s.Sign(sampleClaims(), "k")
			if err != nil {
				t.Fatalf("Sign: %v", err)
			}

			tampered := tamperPayload(t, raw)
			if tampered == raw {
				t.Fatalf("tampered token equals original")
			}

			err = s.Verify(tampered, &jwt.MapClaims{})
			if err == nil {
				t.Fatalf("Verify accepted tampered token")
			}
			if !errors.Is(err, token.ErrSignature) {
				t.Fatalf("errors.Is(err, token.ErrSignature) = false, err = %v", err)
			}
		})
	}
}

func TestSigner_WrongAlgRejected(t *testing.T) {
	t.Parallel()

	es := newES256Signer(t)
	rs := newRS256Signer(t)

	esToken, err := es.Sign(sampleClaims(), "k")
	if err != nil {
		t.Fatalf("ES256 sign: %v", err)
	}
	rsToken, err := rs.Sign(sampleClaims(), "k")
	if err != nil {
		t.Fatalf("RS256 sign: %v", err)
	}

	if err := rs.Verify(esToken, &jwt.MapClaims{}); err == nil {
		t.Fatalf("RS256 signer accepted ES256 token")
	} else if !errors.Is(err, token.ErrSignature) {
		t.Fatalf("errors.Is(err, token.ErrSignature) = false, err = %v", err)
	}
	if err := es.Verify(rsToken, &jwt.MapClaims{}); err == nil {
		t.Fatalf("ES256 signer accepted RS256 token")
	} else if !errors.Is(err, token.ErrSignature) {
		t.Fatalf("errors.Is(err, token.ErrSignature) = false, err = %v", err)
	}
}

func TestNewES256Signer_RejectsBadInputs(t *testing.T) {
	t.Parallel()

	p256, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate P-256: %v", err)
	}
	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("generate P-384: %v", err)
	}

	if _, err := token.NewES256Signer(nil, &p256.PublicKey); err == nil {
		t.Fatalf("expected error for nil private key")
	}
	if _, err := token.NewES256Signer(p256, nil); err == nil {
		t.Fatalf("expected error for nil public key")
	}
	if _, err := token.NewES256Signer(p384, &p384.PublicKey); err == nil {
		t.Fatalf("expected error for non-P256 curve")
	}
}

func TestNewRS256Signer_RejectsSmallKey(t *testing.T) {
	t.Parallel()

	// 1024 bits is below the 2048-bit minimum and must be refused.
	small, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate small RSA key: %v", err)
	}
	if _, err := token.NewRS256Signer(small, &small.PublicKey); err == nil {
		t.Fatalf("expected error for sub-2048-bit RSA key")
	}

	big, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	if _, err := token.NewRS256Signer(nil, &big.PublicKey); err == nil {
		t.Fatalf("expected error for nil private key")
	}
	if _, err := token.NewRS256Signer(big, nil); err == nil {
		t.Fatalf("expected error for nil public key")
	}
}

// tamperPayload flips a byte in the JWT payload segment so the signature no
// longer matches. The result is still a syntactically valid JWT.
func tamperPayload(t *testing.T, raw string) string {
	t.Helper()
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT segments, got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(payload) == 0 {
		t.Fatalf("empty payload")
	}
	payload[0] ^= 0x01
	parts[1] = base64.RawURLEncoding.EncodeToString(payload)
	return strings.Join(parts, ".")
}
