package token_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/menems/saas/pkg/token"
)

// managerClaims is the test-only claims type used across this file. It
// embeds jwt.RegisteredClaims, adds a custom field, and implements
// token.RegisteredClaimsSetter so that Manager.Issue can populate the
// standard fields.
type managerClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
}

func (c *managerClaims) SetRegisteredClaims(rc jwt.RegisteredClaims) {
	c.RegisteredClaims = rc
}

var _ token.Claims = (*managerClaims)(nil)
var _ token.RegisteredClaimsSetter = (*managerClaims)(nil)

func newManagerClaims() *managerClaims { return &managerClaims{} }

// managerTestEnv bundles the artefacts a test needs: signer, key provider,
// and a Manager constructed from them.
type managerTestEnv struct {
	signer  token.Signer
	keys    *token.StaticKeys
	manager *token.Manager[*managerClaims]
	now     time.Time
}

func newES256Env(t *testing.T, mutate func(cfg *token.Config)) managerTestEnv {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA: %v", err)
	}
	signer, err := token.NewES256Signer(priv, &priv.PublicKey)
	if err != nil {
		t.Fatalf("NewES256Signer: %v", err)
	}
	keys, err := token.NewStaticKeys(token.WithSigningKey("es-1", priv))
	if err != nil {
		t.Fatalf("NewStaticKeys: %v", err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	cfg := token.Config{
		Issuer:      "issuer-x",
		Audience:    []string{"aud-a"},
		TTL:         15 * time.Minute,
		Skew:        30 * time.Second,
		KeyProvider: keys,
		Signers:     map[string]token.Signer{"ES256": signer},
		Clock:       func() time.Time { return now },
	}
	if mutate != nil {
		mutate(&cfg)
	}
	mgr, err := token.NewManager[*managerClaims](cfg, newManagerClaims)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return managerTestEnv{signer: signer, keys: keys, manager: mgr, now: now}
}

func TestManager_IssueParseRoundTrip(t *testing.T) {
	t.Parallel()

	env := newES256Env(t, nil)
	in := &managerClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "user-42"},
		UserID:           "user-42",
	}
	raw, err := env.manager.Issue(context.Background(), in)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if raw == "" {
		t.Fatal("Issue returned empty token")
	}

	out, err := env.manager.Parse(context.Background(), raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if out.UserID != "user-42" {
		t.Errorf("UserID = %q, want user-42", out.UserID)
	}
	if out.Issuer != "issuer-x" {
		t.Errorf("Issuer = %q, want issuer-x", out.Issuer)
	}
	if got := []string(out.Audience); len(got) != 1 || got[0] != "aud-a" {
		t.Errorf("Audience = %v, want [aud-a]", got)
	}
	if out.Subject != "user-42" {
		t.Errorf("Subject = %q, want user-42", out.Subject)
	}
	if out.ID == "" {
		t.Error("ID was not populated")
	}
	if out.ExpiresAt == nil || !out.ExpiresAt.Time.Equal(env.now.Add(15*time.Minute)) {
		t.Errorf("ExpiresAt = %v, want %v", out.ExpiresAt, env.now.Add(15*time.Minute))
	}
	if out.NotBefore == nil || !out.NotBefore.Time.Equal(env.now.Add(-30*time.Second)) {
		t.Errorf("NotBefore = %v, want %v", out.NotBefore, env.now.Add(-30*time.Second))
	}
	if out.IssuedAt == nil || !out.IssuedAt.Time.Equal(env.now) {
		t.Errorf("IssuedAt = %v, want %v", out.IssuedAt, env.now)
	}
}

func TestNewManager_ConfigValidation(t *testing.T) {
	t.Parallel()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA: %v", err)
	}
	signer, err := token.NewES256Signer(priv, &priv.PublicKey)
	if err != nil {
		t.Fatalf("NewES256Signer: %v", err)
	}
	keys, err := token.NewStaticKeys(token.WithSigningKey("es-1", priv))
	if err != nil {
		t.Fatalf("NewStaticKeys: %v", err)
	}

	good := func() token.Config {
		return token.Config{
			Issuer:      "issuer-x",
			Audience:    []string{"aud-a"},
			TTL:         time.Minute,
			KeyProvider: keys,
			Signers:     map[string]token.Signer{"ES256": signer},
		}
	}

	cases := []struct {
		name   string
		mutate func(*token.Config)
		want   string
	}{
		{"empty issuer", func(c *token.Config) { c.Issuer = "" }, "Issuer"},
		{"empty audience", func(c *token.Config) { c.Audience = nil }, "Audience"},
		{"blank audience entry", func(c *token.Config) { c.Audience = []string{""} }, "Audience"},
		{"zero ttl", func(c *token.Config) { c.TTL = 0 }, "TTL"},
		{"negative ttl", func(c *token.Config) { c.TTL = -time.Second }, "TTL"},
		{"negative skew", func(c *token.Config) { c.Skew = -time.Second }, "Skew"},
		{"nil key provider", func(c *token.Config) { c.KeyProvider = nil }, "KeyProvider"},
		{"nil signers", func(c *token.Config) { c.Signers = nil }, "Signers"},
		{"empty signers", func(c *token.Config) { c.Signers = map[string]token.Signer{} }, "Signers"},
		{"alg mismatch", func(c *token.Config) { c.Signers = map[string]token.Signer{"RS256": signer} }, "alg"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := good()
			tc.mutate(&cfg)
			_, err := token.NewManager[*managerClaims](cfg, newManagerClaims)
			if err == nil {
				t.Fatalf("NewManager accepted invalid config %#v", cfg)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to mention %q", err, tc.want)
			}
		})
	}

	t.Run("nil claims factory", func(t *testing.T) {
		t.Parallel()
		cfg := good()
		_, err := token.NewManager[*managerClaims](cfg, nil)
		if err == nil {
			t.Fatal("NewManager accepted nil claims factory")
		}
	})
}

func TestManager_Issue_RequiresRegisteredClaimsSetter(t *testing.T) {
	t.Parallel()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA: %v", err)
	}
	signer, err := token.NewES256Signer(priv, &priv.PublicKey)
	if err != nil {
		t.Fatalf("NewES256Signer: %v", err)
	}
	keys, err := token.NewStaticKeys(token.WithSigningKey("es-1", priv))
	if err != nil {
		t.Fatalf("NewStaticKeys: %v", err)
	}
	cfg := token.Config{
		Issuer:      "issuer-x",
		Audience:    []string{"aud-a"},
		TTL:         time.Minute,
		KeyProvider: keys,
		Signers:     map[string]token.Signer{"ES256": signer},
	}
	mgr, err := token.NewManager[jwt.MapClaims](cfg, func() jwt.MapClaims { return jwt.MapClaims{} })
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	_, err = mgr.Issue(context.Background(), jwt.MapClaims{"sub": "x"})
	if err == nil {
		t.Fatal("Issue accepted claims without RegisteredClaimsSetter")
	}
	if !strings.Contains(err.Error(), "RegisteredClaimsSetter") {
		t.Fatalf("error = %v, want it to mention RegisteredClaimsSetter", err)
	}
}

func TestManager_Parse_Expired(t *testing.T) {
	t.Parallel()

	env := newES256Env(t, nil)
	raw, err := env.manager.Issue(context.Background(), &managerClaims{UserID: "u"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// Build a second manager whose clock is well past TTL + Skew.
	future := env.now.Add(time.Hour)
	mgr2, err := token.NewManager[*managerClaims](token.Config{
		Issuer:      "issuer-x",
		Audience:    []string{"aud-a"},
		TTL:         15 * time.Minute,
		Skew:        30 * time.Second,
		KeyProvider: env.keys,
		Signers:     map[string]token.Signer{"ES256": env.signer},
		Clock:       func() time.Time { return future },
	}, newManagerClaims)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	_, err = mgr2.Parse(context.Background(), raw)
	if err == nil {
		t.Fatal("Parse accepted expired token")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("error = %v, want it to mention expiration", err)
	}
}

func TestManager_Parse_NotYetValid(t *testing.T) {
	t.Parallel()

	env := newES256Env(t, nil)
	// Issue at env.now, parse at env.now - 5 minutes (well before nbf-skew).
	past := env.now.Add(-5 * time.Minute)
	raw, err := env.manager.Issue(context.Background(), &managerClaims{UserID: "u"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	mgr2, err := token.NewManager[*managerClaims](token.Config{
		Issuer:      "issuer-x",
		Audience:    []string{"aud-a"},
		TTL:         15 * time.Minute,
		Skew:        30 * time.Second,
		KeyProvider: env.keys,
		Signers:     map[string]token.Signer{"ES256": env.signer},
		Clock:       func() time.Time { return past },
	}, newManagerClaims)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	_, err = mgr2.Parse(context.Background(), raw)
	if err == nil {
		t.Fatal("Parse accepted not-yet-valid token")
	}
	if !strings.Contains(err.Error(), "nbf") && !strings.Contains(err.Error(), "not yet valid") {
		t.Fatalf("error = %v, want it to mention nbf", err)
	}
}

func TestManager_Parse_WrongIssuer(t *testing.T) {
	t.Parallel()

	env := newES256Env(t, nil)
	// Mint a token via a second manager that uses a different issuer.
	other, err := token.NewManager[*managerClaims](token.Config{
		Issuer:      "issuer-other",
		Audience:    []string{"aud-a"},
		TTL:         time.Minute,
		KeyProvider: env.keys,
		Signers:     map[string]token.Signer{"ES256": env.signer},
		Clock:       func() time.Time { return env.now },
	}, newManagerClaims)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	raw, err := other.Issue(context.Background(), &managerClaims{UserID: "u"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	_, err = env.manager.Parse(context.Background(), raw)
	if err == nil {
		t.Fatal("Parse accepted token from another issuer")
	}
	if !strings.Contains(err.Error(), "issuer") {
		t.Fatalf("error = %v, want it to mention issuer", err)
	}
}

func TestManager_Parse_WrongAudience(t *testing.T) {
	t.Parallel()

	env := newES256Env(t, nil)
	other, err := token.NewManager[*managerClaims](token.Config{
		Issuer:      "issuer-x",
		Audience:    []string{"aud-other"},
		TTL:         time.Minute,
		KeyProvider: env.keys,
		Signers:     map[string]token.Signer{"ES256": env.signer},
		Clock:       func() time.Time { return env.now },
	}, newManagerClaims)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	raw, err := other.Issue(context.Background(), &managerClaims{UserID: "u"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	_, err = env.manager.Parse(context.Background(), raw)
	if err == nil {
		t.Fatal("Parse accepted token with wrong audience")
	}
	if !strings.Contains(err.Error(), "audience") {
		t.Fatalf("error = %v, want it to mention audience", err)
	}
}

func TestManager_Parse_UnknownKID(t *testing.T) {
	t.Parallel()

	env := newES256Env(t, nil)

	// Issue a token via a completely separate key provider whose kid the
	// configured manager will not recognise.
	otherPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA: %v", err)
	}
	otherSigner, err := token.NewES256Signer(otherPriv, &otherPriv.PublicKey)
	if err != nil {
		t.Fatalf("NewES256Signer: %v", err)
	}
	otherKeys, err := token.NewStaticKeys(token.WithSigningKey("es-stranger", otherPriv))
	if err != nil {
		t.Fatalf("NewStaticKeys: %v", err)
	}
	otherMgr, err := token.NewManager[*managerClaims](token.Config{
		Issuer:      "issuer-x",
		Audience:    []string{"aud-a"},
		TTL:         time.Minute,
		KeyProvider: otherKeys,
		Signers:     map[string]token.Signer{"ES256": otherSigner},
		Clock:       func() time.Time { return env.now },
	}, newManagerClaims)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	raw, err := otherMgr.Issue(context.Background(), &managerClaims{UserID: "u"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	_, err = env.manager.Parse(context.Background(), raw)
	if err == nil {
		t.Fatal("Parse accepted token with unknown kid")
	}
	if !strings.Contains(err.Error(), "kid") && !strings.Contains(err.Error(), "unknown kid") {
		t.Fatalf("error = %v, want it to mention kid", err)
	}
}

func TestManager_Parse_TamperedSignature(t *testing.T) {
	t.Parallel()

	env := newES256Env(t, nil)
	raw, err := env.manager.Issue(context.Background(), &managerClaims{UserID: "u"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	payload[0] ^= 0x01
	parts[1] = base64.RawURLEncoding.EncodeToString(payload)
	tampered := strings.Join(parts, ".")

	_, err = env.manager.Parse(context.Background(), tampered)
	if err == nil {
		t.Fatal("Parse accepted tampered token")
	}
}

func TestManager_Parse_MissingKIDHeader(t *testing.T) {
	t.Parallel()

	env := newES256Env(t, nil)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA: %v", err)
	}
	// Build a token via jwt directly, omitting the kid header.
	rc := jwt.RegisteredClaims{
		Issuer:    "issuer-x",
		Subject:   "user-1",
		Audience:  jwt.ClaimStrings{"aud-a"},
		IssuedAt:  jwt.NewNumericDate(env.now),
		NotBefore: jwt.NewNumericDate(env.now),
		ExpiresAt: jwt.NewNumericDate(env.now.Add(time.Minute)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, &managerClaims{RegisteredClaims: rc})
	delete(tok.Header, "kid")
	raw, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	_, err = env.manager.Parse(context.Background(), raw)
	if err == nil {
		t.Fatal("Parse accepted token without kid header")
	}
	if !strings.Contains(err.Error(), "kid") {
		t.Fatalf("error = %v, want it to mention kid", err)
	}
}

func TestManager_Parse_MissingAlgHeader(t *testing.T) {
	t.Parallel()

	env := newES256Env(t, nil)
	// Hand-craft a token whose header has no alg field. Use the existing
	// kid so that the manager passes kid resolution before failing on alg.
	header := map[string]any{"typ": "JWT", "kid": "es-1"}
	payload := map[string]any{
		"iss": "issuer-x",
		"aud": []string{"aud-a"},
		"iat": env.now.Unix(),
		"nbf": env.now.Unix(),
		"exp": env.now.Add(time.Minute).Unix(),
	}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)
	raw := base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(pb) + ".AA"

	_, err := env.manager.Parse(context.Background(), raw)
	if err == nil {
		t.Fatal("Parse accepted token without alg header")
	}
	if !strings.Contains(err.Error(), "alg") {
		t.Fatalf("error = %v, want it to mention alg", err)
	}
}

func TestManager_RS256_RoundTrip(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA: %v", err)
	}
	signer, err := token.NewRS256Signer(priv, &priv.PublicKey)
	if err != nil {
		t.Fatalf("NewRS256Signer: %v", err)
	}
	keys, err := token.NewStaticKeys(token.WithSigningKey("rs-1", priv))
	if err != nil {
		t.Fatalf("NewStaticKeys: %v", err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	mgr, err := token.NewManager[*managerClaims](token.Config{
		Issuer:      "issuer-rs",
		Audience:    []string{"aud-rs"},
		TTL:         time.Minute,
		Skew:        time.Second,
		KeyProvider: keys,
		Signers:     map[string]token.Signer{"RS256": signer},
		Clock:       func() time.Time { return now },
	}, newManagerClaims)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	raw, err := mgr.Issue(context.Background(), &managerClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "user-rs"},
		UserID:           "user-rs",
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	out, err := mgr.Parse(context.Background(), raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if out.UserID != "user-rs" {
		t.Errorf("UserID = %q, want user-rs", out.UserID)
	}
	if out.Subject != "user-rs" {
		t.Errorf("Subject = %q, want user-rs", out.Subject)
	}

	// Confirm the alg header is RS256 to prove the path is genuinely RSA.
	parsed, _, err := jwt.NewParser().ParseUnverified(raw, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("ParseUnverified: %v", err)
	}
	if got, _ := parsed.Header["alg"].(string); got != "RS256" {
		t.Fatalf("alg = %q, want RS256", got)
	}
}

func TestManager_ConcurrentIssueParse(t *testing.T) {
	t.Parallel()

	env := newES256Env(t, nil)
	done := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			raw, err := env.manager.Issue(context.Background(), &managerClaims{UserID: "u"})
			if err != nil {
				done <- err
				return
			}
			if _, err := env.manager.Parse(context.Background(), raw); err != nil {
				done <- err
				return
			}
			done <- nil
		}()
	}
	for i := 0; i < 8; i++ {
		if err := <-done; err != nil {
			t.Fatalf("goroutine: %v", err)
		}
	}
}
