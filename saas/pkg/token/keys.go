package token

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// KeyProvider resolves the asymmetric keys used to sign and verify tokens.
//
// Implementations must be safe for concurrent use. Lookups are keyed by the
// JWT `kid` header so that verification can locate the public key that
// matches whichever signing key was active when the token was issued.
type KeyProvider interface {
	// SigningKey returns the currently active private key together with the
	// kid that callers should embed in the JWT header.
	SigningKey() (kid string, key crypto.Signer, err error)

	// VerificationKey returns the public key registered under kid. It must
	// return a non-nil error if kid is unknown; callers rely on that error
	// to differentiate "no such key" from "signature mismatch".
	VerificationKey(kid string) (crypto.PublicKey, error)
}

// StaticKeys is an in-memory [KeyProvider] backed by a fixed set of keys.
// It is intended for processes that load their keys once at startup; runtime
// rotation is out of scope for this package.
type StaticKeys struct {
	signingKID string
	signingKey crypto.Signer
	verifiers  map[string]crypto.PublicKey
}

// Option configures a [StaticKeys] instance.
type Option func(*staticKeysBuilder) error

type staticKeysBuilder struct {
	signingKID string
	signingKey crypto.Signer
	verifiers  map[string]crypto.PublicKey
}

// WithSigningKey designates the private key used by [StaticKeys.SigningKey].
// The matching public key is automatically registered under the same kid for
// verification, so callers do not have to add it twice.
func WithSigningKey(kid string, key crypto.Signer) Option {
	return func(b *staticKeysBuilder) error {
		if kid == "" {
			return errors.New("token: signing kid must not be empty")
		}
		if key == nil {
			return errors.New("token: signing key must not be nil")
		}
		if err := checkSupportedSigner(key); err != nil {
			return err
		}
		b.signingKID = kid
		b.signingKey = key
		b.verifiers[kid] = key.Public()
		return nil
	}
}

// WithVerificationKey registers an additional public key for verification
// under kid. Use this to accept tokens signed by previous signing keys during
// a rotation window.
func WithVerificationKey(kid string, key crypto.PublicKey) Option {
	return func(b *staticKeysBuilder) error {
		if kid == "" {
			return errors.New("token: verification kid must not be empty")
		}
		if err := checkSupportedPublicKey(key); err != nil {
			return err
		}
		b.verifiers[kid] = key
		return nil
	}
}

// NewStaticKeys assembles a [StaticKeys] from the given options. Exactly one
// [WithSigningKey] option must be supplied.
func NewStaticKeys(opts ...Option) (*StaticKeys, error) {
	b := &staticKeysBuilder{verifiers: make(map[string]crypto.PublicKey)}
	for _, opt := range opts {
		if err := opt(b); err != nil {
			return nil, err
		}
	}
	if b.signingKey == nil {
		return nil, errors.New("token: NewStaticKeys requires WithSigningKey")
	}
	return &StaticKeys{
		signingKID: b.signingKID,
		signingKey: b.signingKey,
		verifiers:  b.verifiers,
	}, nil
}

// SigningKey implements [KeyProvider].
func (s *StaticKeys) SigningKey() (string, crypto.Signer, error) {
	return s.signingKID, s.signingKey, nil
}

// VerificationKey implements [KeyProvider]. It returns [ErrUnknownKID]
// (wrapped) when kid is not registered.
func (s *StaticKeys) VerificationKey(kid string) (crypto.PublicKey, error) {
	key, ok := s.verifiers[kid]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKID, kid)
	}
	return key, nil
}

// ParsePrivateKeyPEM decodes a PEM-encoded private key. It accepts the
// PKCS#8 ("PRIVATE KEY"), PKCS#1 ("RSA PRIVATE KEY") and SEC1
// ("EC PRIVATE KEY") encodings. Only RSA and ECDSA P-256 keys are accepted.
func ParsePrivateKeyPEM(data []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("token: no PEM block found in private key data")
	}
	var (
		key any
		err error
	)
	switch block.Type {
	case "PRIVATE KEY":
		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("token: unsupported private key PEM type %q", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("token: parse private key: %w", err)
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("token: parsed key of type %T is not a crypto.Signer", key)
	}
	if err := checkSupportedSigner(signer); err != nil {
		return nil, err
	}
	return signer, nil
}

// ParsePublicKeyPEM decodes a PEM-encoded public key in PKIX
// ("PUBLIC KEY") form. Only RSA and ECDSA P-256 keys are accepted.
func ParsePublicKeyPEM(data []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("token: no PEM block found in public key data")
	}
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("token: unsupported public key PEM type %q", block.Type)
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("token: parse public key: %w", err)
	}
	if err := checkSupportedPublicKey(key); err != nil {
		return nil, err
	}
	return key, nil
}

// LoadStaticKeysFromPEM builds a [StaticKeys] from in-memory PEM material.
// signingKID names the entry in signingKeyPEM that should be used for
// signing; verificationKeysPEM provides additional kid→public-key PEM
// mappings (typically previous signing keys retained for verification).
func LoadStaticKeysFromPEM(signingKID string, signingKeyPEM []byte, verificationKeysPEM map[string][]byte) (*StaticKeys, error) {
	signer, err := ParsePrivateKeyPEM(signingKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("token: signing key %q: %w", signingKID, err)
	}
	opts := []Option{WithSigningKey(signingKID, signer)}
	for kid, pemBytes := range verificationKeysPEM {
		pub, err := ParsePublicKeyPEM(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("token: verification key %q: %w", kid, err)
		}
		opts = append(opts, WithVerificationKey(kid, pub))
	}
	return NewStaticKeys(opts...)
}

// LoadStaticKeysFromFiles is the on-disk counterpart of
// [LoadStaticKeysFromPEM]. Each path is read from the filesystem before
// being parsed.
func LoadStaticKeysFromFiles(signingKID, signingKeyPath string, verificationKeyPaths map[string]string) (*StaticKeys, error) {
	signingPEM, err := os.ReadFile(signingKeyPath)
	if err != nil {
		return nil, fmt.Errorf("token: read signing key %q: %w", signingKeyPath, err)
	}
	verificationPEMs := make(map[string][]byte, len(verificationKeyPaths))
	for kid, path := range verificationKeyPaths {
		pemBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("token: read verification key %q: %w", path, err)
		}
		verificationPEMs[kid] = pemBytes
	}
	return LoadStaticKeysFromPEM(signingKID, signingPEM, verificationPEMs)
}

func checkSupportedSigner(s crypto.Signer) error {
	switch k := s.(type) {
	case *rsa.PrivateKey:
		return nil
	case *ecdsa.PrivateKey:
		if k.Curve != elliptic.P256() {
			return fmt.Errorf("token: unsupported EC curve %q (only P-256 is accepted)", k.Curve.Params().Name)
		}
		return nil
	default:
		return fmt.Errorf("token: unsupported signing key type %T (want RSA or ECDSA P-256)", s)
	}
}

func checkSupportedPublicKey(k crypto.PublicKey) error {
	switch k := k.(type) {
	case *rsa.PublicKey:
		return nil
	case *ecdsa.PublicKey:
		if k.Curve != elliptic.P256() {
			return fmt.Errorf("token: unsupported EC curve %q (only P-256 is accepted)", k.Curve.Params().Name)
		}
		return nil
	default:
		return fmt.Errorf("token: unsupported public key type %T (want RSA or ECDSA P-256)", k)
	}
}
