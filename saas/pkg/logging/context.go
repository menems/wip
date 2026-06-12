package logging

import (
	"context"
	"log/slog"
)

type contextKey struct{}

// WithLogger returns a new context carrying log.
func WithLogger(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, log)
}

// FromContext retrieves the *slog.Logger stored by WithLogger.
// If no logger is present it falls back to slog.Default().
func FromContext(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(contextKey{}).(*slog.Logger); ok && log != nil {
		return log
	}
	return slog.Default()
}
