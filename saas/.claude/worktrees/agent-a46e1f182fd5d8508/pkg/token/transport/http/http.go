// Package tokenhttp exposes a [token.Manager] as plain net/http middleware
// for endpoints that sit outside a ConnectRPC handler.
//
// Two middlewares are provided. [RequireAuth] only enforces that the request
// carries a valid bearer token; [Require] also checks an application-supplied
// predicate against the parsed claims. Both stash the typed claims in the
// request context so that downstream handlers can retrieve them via
// [ClaimsFromContext].
package tokenhttp

import (
	"context"
	"net/http"
	"strings"

	"github.com/menems/saas/pkg/token"
)

// ctxKey is a generic, unexported type used as the context key under which the
// parsed claims of type T are stored. Using a generic struct as the key keeps
// the storage type-safe without a package-global registry.
type ctxKey[T token.Claims] struct{}

// RequireAuth returns middleware that rejects any request without a valid
// bearer token. The Authorization header must be of the form
// "Bearer <jwt>"; on success, the parsed claims of type T are placed into
// the request context and the next handler is invoked. On any failure
// (missing/malformed header, invalid token) the response is
// 401 Unauthorized with a "WWW-Authenticate: Bearer" challenge header.
func RequireAuth[T token.Claims](m *token.Manager[T]) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := authenticate(w, r, m)
			if !ok {
				return
			}
			next.ServeHTTP(w, r.WithContext(withClaims[T](r.Context(), claims)))
		})
	}
}

// Require returns middleware that wraps [RequireAuth] with an additional
// authorization step: after the token is verified, predicate is called with
// the parsed claims. A false return produces 403 Forbidden; a true return
// delegates to next. predicate must not be nil.
func Require[T token.Claims](m *token.Manager[T], predicate func(T) bool) func(http.Handler) http.Handler {
	if predicate == nil {
		panic("tokenhttp: Require called with nil predicate")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := authenticate(w, r, m)
			if !ok {
				return
			}
			if !predicate(claims) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r.WithContext(withClaims[T](r.Context(), claims)))
		})
	}
}

// ClaimsFromContext returns the claims of type T previously stored by
// [RequireAuth] or [Require]. The second return value is false when no claims
// of that type are present, which lets handlers distinguish "anonymous"
// from "authenticated but with a zero value".
func ClaimsFromContext[T token.Claims](ctx context.Context) (T, bool) {
	v, ok := ctx.Value(ctxKey[T]{}).(T)
	if !ok {
		var zero T
		return zero, false
	}
	return v, true
}

// authenticate centralises the bearer-token check shared by both middlewares.
// It writes the 401 response itself when the header is missing/malformed or
// the token fails to parse; callers only need to check the boolean.
func authenticate[T token.Claims](w http.ResponseWriter, r *http.Request, m *token.Manager[T]) (T, bool) {
	var zero T
	raw, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return zero, false
	}
	claims, err := m.Parse(r.Context(), raw)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return zero, false
	}
	return claims, true
}

// bearerToken extracts the credentials from an Authorization header value.
// It returns false unless the scheme is exactly "Bearer" (case-insensitive)
// and the credentials are non-empty.
func bearerToken(header string) (string, bool) {
	const scheme = "Bearer"
	if len(header) <= len(scheme)+1 {
		return "", false
	}
	if !strings.EqualFold(header[:len(scheme)], scheme) {
		return "", false
	}
	if header[len(scheme)] != ' ' {
		return "", false
	}
	raw := strings.TrimSpace(header[len(scheme)+1:])
	if raw == "" {
		return "", false
	}
	return raw, true
}

// withClaims is the type-safe writer counterpart of [ClaimsFromContext].
func withClaims[T token.Claims](ctx context.Context, c T) context.Context {
	return context.WithValue(ctx, ctxKey[T]{}, c)
}
