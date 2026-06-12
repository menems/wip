package logging

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)

// Interceptor is a ConnectRPC interceptor that logs every RPC call via slog.
type Interceptor struct {
	log *slog.Logger
}

// NewInterceptor creates a new logging interceptor using the provided logger.
func NewInterceptor(log *slog.Logger) *Interceptor {
	return &Interceptor{log: log}
}

// WrapUnary implements connect.Interceptor.
func (i *Interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		procedure := req.Spec().Procedure

		ctx = WithLogger(ctx, i.log)
		resp, err := next(ctx, req)

		i.log.InfoContext(ctx, "rpc",
			slog.String("procedure", procedure),
			slog.Duration("duration", time.Since(start)),
			slog.String("code", codeString(codeOf(err))),
		)
		return resp, err
	}
}

// WrapStreamingClient implements connect.Interceptor (no-op for client streams).
func (i *Interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor.
func (i *Interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		procedure := conn.Spec().Procedure

		ctx = WithLogger(ctx, i.log)
		err := next(ctx, conn)

		i.log.InfoContext(ctx, "rpc",
			slog.String("procedure", procedure),
			slog.Duration("duration", time.Since(start)),
			slog.String("code", codeString(codeOf(err))),
		)
		return err
	}
}

// codeOK is code 0 (success). connect does not export a CodeOK constant.
const codeOK connect.Code = 0

// codeOf extracts the connect.Code from err. Returns codeOK (0) when err is
// nil and CodeUnknown when err is non-nil but not a *connect.Error.
func codeOf(err error) connect.Code {
	if err == nil {
		return codeOK
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return connectErr.Code()
	}
	return connect.CodeUnknown
}

// codeString returns a human-readable name for c. It returns "ok" for code 0
// because connect.Code.String() returns "code_0" for the zero value.
func codeString(c connect.Code) string {
	if c == codeOK {
		return "ok"
	}
	return c.String()
}
