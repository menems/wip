package tokenhttp_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/menems/saas/pkg/token"
	tokenhttp "github.com/menems/saas/pkg/token/transport/http"
)

// testClaims mirrors the claims type used by the manager unit tests: it
// embeds jwt.RegisteredClaims, exposes a custom field, and implements
// token.RegisteredClaimsSetter so that Manager.Issue can fill in the
// registered fields.
type testClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
	Role   string `json:"role"`
}

func (c *testClaims) SetRegisteredClaims(rc jwt.RegisteredClaims) {
	c.RegisteredClaims = rc
}

func newTestClaims() *testClaims { return &testClaims{} }

// testEnv bundles a Manager wired to in-process ES256 keys plus the frozen
// clock the manager uses, so tests can compute deterministic expectations.
type testEnv struct {
	manager *token.Manager[*testClaims]
	signer  token.Signer
	keys    *token.StaticKeys
	now     time.Time
}

func newEnv(t *testing.T) testEnv {
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
	mgr, err := token.NewManager[*testClaims](token.Config{
		Issuer:      "issuer-x",
		Audience:    []string{"aud-a"},
		TTL:         15 * time.Minute,
		Skew:        30 * time.Second,
		KeyProvider: keys,
		Signers:     map[string]token.Signer{"ES256": signer},
		Clock:       func() time.Time { return now },
	}, newTestClaims)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return testEnv{manager: mgr, signer: signer, keys: keys, now: now}
}

func issueToken(t *testing.T, env testEnv, c *testClaims) string {
	t.Helper()
	raw, err := env.manager.Issue(context.Background(), c)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	return raw
}

// expiredEnv issues with one manager and parses with another whose clock is
// well past TTL+Skew, which guarantees Parse rejects the token as expired.
func expiredToken(t *testing.T, env testEnv) string {
	t.Helper()
	return issueToken(t, env, &testClaims{UserID: "u", Role: "user"})
}

func sinkHandler(seen **testClaims) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := tokenhttp.ClaimsFromContext[*testClaims](r.Context())
		if ok {
			*seen = c
		}
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var seen *testClaims
	srv := httptest.NewServer(tokenhttp.RequireAuth(env.manager)(sinkHandler(&seen)))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if got := resp.Header.Get("WWW-Authenticate"); got != "Bearer" {
		t.Errorf("WWW-Authenticate = %q, want Bearer", got)
	}
	if seen != nil {
		t.Error("handler ran despite missing Authorization header")
	}
}

func TestRequireAuth_MalformedHeader(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var seen *testClaims
	srv := httptest.NewServer(tokenhttp.RequireAuth(env.manager)(sinkHandler(&seen)))
	t.Cleanup(srv.Close)

	cases := []struct {
		name   string
		header string
	}{
		{"basic scheme", "Basic dXNlcjpwYXNz"},
		{"bearer no token", "Bearer "},
		{"bare token", "abcdef"},
		{"bearer no space", "Bearerabc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			req.Header.Set("Authorization", tc.header)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Do: %v", err)
			}
			t.Cleanup(func() { _ = resp.Body.Close() })

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
			}
		})
	}
}

func TestRequireAuth_ParseFailure(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	raw := expiredToken(t, env)

	// Build a second manager whose clock is well past TTL+Skew so that
	// Parse rejects the freshly-issued token as expired.
	future := env.now.Add(time.Hour)
	mgr2, err := token.NewManager[*testClaims](token.Config{
		Issuer:      "issuer-x",
		Audience:    []string{"aud-a"},
		TTL:         15 * time.Minute,
		Skew:        30 * time.Second,
		KeyProvider: env.keys,
		Signers:     map[string]token.Signer{"ES256": env.signer},
		Clock:       func() time.Time { return future },
	}, newTestClaims)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	var seen *testClaims
	srv := httptest.NewServer(tokenhttp.RequireAuth(mgr2)(sinkHandler(&seen)))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if got := resp.Header.Get("WWW-Authenticate"); got != "Bearer" {
		t.Errorf("WWW-Authenticate = %q, want Bearer", got)
	}
	if seen != nil {
		t.Error("handler ran despite invalid token")
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	raw := issueToken(t, env, &testClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "user-7"},
		UserID:           "user-7",
		Role:             "admin",
	})

	var seen *testClaims
	srv := httptest.NewServer(tokenhttp.RequireAuth(env.manager)(sinkHandler(&seen)))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if seen == nil {
		t.Fatal("handler did not run / claims missing from context")
	}
	if seen.UserID != "user-7" {
		t.Errorf("UserID = %q, want user-7", seen.UserID)
	}
	if seen.Role != "admin" {
		t.Errorf("Role = %q, want admin", seen.Role)
	}
	if seen.Subject != "user-7" {
		t.Errorf("Subject = %q, want user-7", seen.Subject)
	}
}

func TestRequire_PredicateRejects(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	raw := issueToken(t, env, &testClaims{UserID: "u", Role: "user"})

	adminOnly := func(c *testClaims) bool { return c.Role == "admin" }

	var seen *testClaims
	srv := httptest.NewServer(tokenhttp.Require(env.manager, adminOnly)(sinkHandler(&seen)))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	if seen != nil {
		t.Error("handler ran despite predicate rejection")
	}
}

func TestRequire_PredicatePasses(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	raw := issueToken(t, env, &testClaims{UserID: "u", Role: "admin"})

	adminOnly := func(c *testClaims) bool { return c.Role == "admin" }

	var seen *testClaims
	srv := httptest.NewServer(tokenhttp.Require(env.manager, adminOnly)(sinkHandler(&seen)))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if seen == nil {
		t.Fatal("handler did not run despite passing predicate")
	}
	if seen.Role != "admin" {
		t.Errorf("Role = %q, want admin", seen.Role)
	}
}

func TestClaimsFromContext_Missing(t *testing.T) {
	t.Parallel()

	if c, ok := tokenhttp.ClaimsFromContext[*testClaims](context.Background()); ok || c != nil {
		t.Fatalf("ClaimsFromContext on empty context = (%v, %v), want (nil, false)", c, ok)
	}
}
