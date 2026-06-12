package auth

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
)

// NewInterceptor returns a Connect interceptor that validates Bearer tokens.
// It extracts the token from the Authorization header, looks up the user ID,
// and injects it into the context. Returns CodeUnauthenticated on failure.
func NewInterceptor(l TokenLookup) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token, ok := bearerToken(req.Header().Get("Authorization"))
			if !ok {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing or invalid Authorization header"))
			}
			userID, err := l.FindUserByToken(ctx, token)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid token"))
			}
			return next(WithUserID(ctx, userID), req)
		}
	})
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := header[len(prefix):]
	if token == "" {
		return "", false
	}
	return token, true
}
