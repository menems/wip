package authz

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
)

// TokenVerifier validates a JWT token and returns the embedded claims.
type TokenVerifier interface {
	VerifyToken(ctx context.Context, token string) (Claims, error)
}

// Interceptor is a ConnectRPC unary interceptor that extracts a Bearer token
// from the Authorization header, validates it, and injects claims into context.
type Interceptor struct {
	verifier   TokenVerifier
	publicProc map[string]struct{}
}

// Option configures the auth interceptor.
type Option func(*Interceptor)

// WithPublicProcedure marks a procedure as public (no auth required).
func WithPublicProcedure(procedure string) Option {
	return func(i *Interceptor) {
		i.publicProc[procedure] = struct{}{}
	}
}

// NewInterceptor creates a new auth interceptor.
func NewInterceptor(verifier TokenVerifier, opts ...Option) *Interceptor {
	i := &Interceptor{
		verifier:   verifier,
		publicProc: make(map[string]struct{}),
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// WrapUnary implements connect.Interceptor.
func (i *Interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		procedure := req.Spec().Procedure
		_, isPublic := i.publicProc[procedure]

		token, ok := extractBearerToken(req.Header().Get("Authorization"))
		if !ok {
			if isPublic {
				return next(ctx, req)
			}
			return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing or malformed authorization header"))
		}

		claims, err := i.verifier.VerifyToken(ctx, token)
		if err != nil {
			if isPublic {
				return next(ctx, req)
			}
			return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token: %w", err))
		}

		ctx = ContextWithClaims(ctx, claims)
		return next(ctx, req)
	}
}

// WrapStreamingClient implements connect.Interceptor (no-op for client streams).
func (i *Interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor.
func (i *Interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		procedure := conn.Spec().Procedure
		_, isPublic := i.publicProc[procedure]

		token, ok := extractBearerToken(conn.RequestHeader().Get("Authorization"))
		if !ok {
			if isPublic {
				return next(ctx, conn)
			}
			return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing or malformed authorization header"))
		}

		claims, err := i.verifier.VerifyToken(ctx, token)
		if err != nil {
			if isPublic {
				return next(ctx, conn)
			}
			return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token: %w", err))
		}

		ctx = ContextWithClaims(ctx, claims)
		return next(ctx, conn)
	}
}

// extractBearerToken parses a "Bearer <token>" string.
func extractBearerToken(header string) (string, bool) {
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
