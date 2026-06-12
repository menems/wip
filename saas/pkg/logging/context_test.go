package logging_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/menems/saas/pkg/logging"
)

func TestFromContext_RoundTrip(t *testing.T) {
	t.Parallel()

	h := &fakeHandler{}
	log := slog.New(h)

	ctx := logging.WithLogger(context.Background(), log)
	got := logging.FromContext(ctx)

	if got != log {
		t.Errorf("FromContext: got %p, want %p", got, log)
	}
}

func TestFromContext_Fallback(t *testing.T) {
	t.Parallel()

	got := logging.FromContext(context.Background())

	if got != slog.Default() {
		t.Errorf("FromContext fallback: got %p, want slog.Default() %p", got, slog.Default())
	}
}
