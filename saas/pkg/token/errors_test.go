package token_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/menems/saas/pkg/token"
)

// Compile-time checks: every sentinel must satisfy the error interface and be
// exported (referencing them here fails to build if any rename occurs).
var (
	_ error = token.ErrInvalid
	_ error = token.ErrExpired
	_ error = token.ErrWrongAudience
	_ error = token.ErrWrongIssuer
	_ error = token.ErrUnknownKID
	_ error = token.ErrSignature
)

func TestSentinelsHaveTokenPrefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{"ErrInvalid", token.ErrInvalid},
		{"ErrExpired", token.ErrExpired},
		{"ErrWrongAudience", token.ErrWrongAudience},
		{"ErrWrongIssuer", token.ErrWrongIssuer},
		{"ErrUnknownKID", token.ErrUnknownKID},
		{"ErrSignature", token.ErrSignature},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !strings.HasPrefix(tc.err.Error(), "token: ") {
				t.Fatalf("%s.Error() = %q, want \"token: \" prefix", tc.name, tc.err.Error())
			}
		})
	}
}

func TestSentinelsAreDistinct(t *testing.T) {
	t.Parallel()

	all := []struct {
		name string
		err  error
	}{
		{"ErrInvalid", token.ErrInvalid},
		{"ErrExpired", token.ErrExpired},
		{"ErrWrongAudience", token.ErrWrongAudience},
		{"ErrWrongIssuer", token.ErrWrongIssuer},
		{"ErrUnknownKID", token.ErrUnknownKID},
		{"ErrSignature", token.ErrSignature},
	}

	for i, a := range all {
		for j, b := range all {
			if i == j {
				continue
			}
			if errors.Is(a.err, b.err) {
				t.Fatalf("errors.Is(%s, %s) = true; sentinels must be distinct", a.name, b.name)
			}
		}
	}
}

func TestErrUnknownKID_WrapsThroughVerificationKey(t *testing.T) {
	t.Parallel()

	priv := mustGenerateECDSA(t)
	pemBytes := mustMarshalPKCS8(t, priv)

	keys, err := token.LoadStaticKeysFromPEM("active", pemBytes, nil)
	if err != nil {
		t.Fatalf("LoadStaticKeysFromPEM: %v", err)
	}

	_, err = keys.VerificationKey("missing")
	if err == nil {
		t.Fatal("VerificationKey returned nil error for unknown kid")
	}
	if !errors.Is(err, token.ErrUnknownKID) {
		t.Fatalf("errors.Is(err, token.ErrUnknownKID) = false, err = %v", err)
	}
}
