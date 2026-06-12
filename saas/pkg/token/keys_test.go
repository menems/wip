package token_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/menems/saas/pkg/token"
)

func TestStaticKeys_ES256_RoundTrip(t *testing.T) {
	t.Parallel()
	priv := mustGenerateECDSA(t)
	pemBytes := mustMarshalPKCS8(t, priv)

	keys, err := token.LoadStaticKeysFromPEM("es-1", pemBytes, nil)
	if err != nil {
		t.Fatalf("LoadStaticKeysFromPEM: %v", err)
	}

	kid, signer, err := keys.SigningKey()
	if err != nil {
		t.Fatalf("SigningKey: %v", err)
	}
	if kid != "es-1" {
		t.Fatalf("kid = %q, want es-1", kid)
	}
	if _, ok := signer.(*ecdsa.PrivateKey); !ok {
		t.Fatalf("signer type = %T, want *ecdsa.PrivateKey", signer)
	}

	pub, err := keys.VerificationKey("es-1")
	if err != nil {
		t.Fatalf("VerificationKey: %v", err)
	}
	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("pub type = %T, want *ecdsa.PublicKey", pub)
	}
	if !ecPub.Equal(&priv.PublicKey) {
		t.Fatalf("verification key does not match signing key")
	}
}

func TestStaticKeys_RS256_RoundTrip(t *testing.T) {
	t.Parallel()
	priv := mustGenerateRSA(t)
	pemBytes := mustMarshalPKCS8(t, priv)

	keys, err := token.LoadStaticKeysFromPEM("rs-1", pemBytes, nil)
	if err != nil {
		t.Fatalf("LoadStaticKeysFromPEM: %v", err)
	}

	kid, signer, err := keys.SigningKey()
	if err != nil {
		t.Fatalf("SigningKey: %v", err)
	}
	if kid != "rs-1" {
		t.Fatalf("kid = %q, want rs-1", kid)
	}
	if _, ok := signer.(*rsa.PrivateKey); !ok {
		t.Fatalf("signer type = %T, want *rsa.PrivateKey", signer)
	}

	pub, err := keys.VerificationKey("rs-1")
	if err != nil {
		t.Fatalf("VerificationKey: %v", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("pub type = %T, want *rsa.PublicKey", pub)
	}
	if !rsaPub.Equal(&priv.PublicKey) {
		t.Fatalf("verification key does not match signing key")
	}
}

func TestStaticKeys_UnknownKID(t *testing.T) {
	t.Parallel()
	priv := mustGenerateECDSA(t)
	pemBytes := mustMarshalPKCS8(t, priv)

	keys, err := token.LoadStaticKeysFromPEM("es-1", pemBytes, nil)
	if err != nil {
		t.Fatalf("LoadStaticKeysFromPEM: %v", err)
	}

	_, err = keys.VerificationKey("nope")
	if err == nil {
		t.Fatal("VerificationKey returned nil error for unknown kid")
	}
	if !errors.Is(err, token.ErrUnknownKID) {
		t.Fatalf("errors.Is(err, token.ErrUnknownKID) = false, err = %v", err)
	}
	if !strings.Contains(err.Error(), `"nope"`) {
		t.Fatalf("error = %v, want it to include the requested kid", err)
	}
}

func TestStaticKeys_AdditionalVerificationKey(t *testing.T) {
	t.Parallel()
	signing := mustGenerateECDSA(t)
	rotated := mustGenerateECDSA(t)

	signingPEM := mustMarshalPKCS8(t, signing)
	rotatedPEM := mustMarshalPKIX(t, &rotated.PublicKey)

	keys, err := token.LoadStaticKeysFromPEM("active", signingPEM, map[string][]byte{
		"previous": rotatedPEM,
	})
	if err != nil {
		t.Fatalf("LoadStaticKeysFromPEM: %v", err)
	}

	if _, err := keys.VerificationKey("active"); err != nil {
		t.Fatalf("VerificationKey(active): %v", err)
	}
	if _, err := keys.VerificationKey("previous"); err != nil {
		t.Fatalf("VerificationKey(previous): %v", err)
	}
}

func TestParsePrivateKeyPEM_MalformedPEM(t *testing.T) {
	t.Parallel()
	_, err := token.ParsePrivateKeyPEM([]byte("not a pem block"))
	if err == nil {
		t.Fatal("ParsePrivateKeyPEM accepted garbage input")
	}
}

func TestParsePrivateKeyPEM_RejectsEd25519(t *testing.T) {
	t.Parallel()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519: %v", err)
	}
	pemBytes := mustMarshalPKCS8(t, priv)

	_, err = token.ParsePrivateKeyPEM(pemBytes)
	if err == nil {
		t.Fatal("ParsePrivateKeyPEM accepted ed25519 key")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("error = %v, want it to mention unsupported key type", err)
	}
}

func TestParsePrivateKeyPEM_RejectsNonP256Curve(t *testing.T) {
	t.Parallel()
	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("generate P-384: %v", err)
	}
	pemBytes := mustMarshalPKCS8(t, priv)

	_, err = token.ParsePrivateKeyPEM(pemBytes)
	if err == nil {
		t.Fatal("ParsePrivateKeyPEM accepted P-384 EC key")
	}
	if !strings.Contains(err.Error(), "P-256") {
		t.Fatalf("error = %v, want it to mention P-256", err)
	}
}

func TestNewStaticKeys_RequiresSigningKey(t *testing.T) {
	t.Parallel()
	_, err := token.NewStaticKeys()
	if err == nil {
		t.Fatal("NewStaticKeys() accepted zero options")
	}
}

func TestLoadStaticKeysFromFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	signing := mustGenerateECDSA(t)
	rotated := mustGenerateRSA(t)

	signingPath := filepath.Join(dir, "signing.pem")
	if err := os.WriteFile(signingPath, mustMarshalPKCS8(t, signing), 0o600); err != nil {
		t.Fatalf("write signing: %v", err)
	}
	rotatedPath := filepath.Join(dir, "rotated.pub.pem")
	if err := os.WriteFile(rotatedPath, mustMarshalPKIX(t, &rotated.PublicKey), 0o600); err != nil {
		t.Fatalf("write rotated: %v", err)
	}

	keys, err := token.LoadStaticKeysFromFiles("active", signingPath, map[string]string{
		"previous": rotatedPath,
	})
	if err != nil {
		t.Fatalf("LoadStaticKeysFromFiles: %v", err)
	}
	if _, err := keys.VerificationKey("previous"); err != nil {
		t.Fatalf("VerificationKey(previous): %v", err)
	}
	kid, _, err := keys.SigningKey()
	if err != nil {
		t.Fatalf("SigningKey: %v", err)
	}
	if kid != "active" {
		t.Fatalf("kid = %q, want active", kid)
	}
}

// helpers

func mustGenerateECDSA(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA: %v", err)
	}
	return priv
}

func mustGenerateRSA(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	// 2048 is the smallest size RS256 accepts in practice. Larger sizes are
	// slow under -race; keep tests fast.
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA: %v", err)
	}
	return priv
}

func mustMarshalPKCS8(t *testing.T, key any) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal PKCS8: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func mustMarshalPKIX(t *testing.T, pub any) []byte {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal PKIX: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
}
