# slog-interceptor
> New pkg/logging package — ConnectRPC interceptor that logs every RPC call via slog

**Created**: 2026-03-12 | **Branch**: feat/slog-interceptor

## Steps
1. [backend] Create pkg/logging with NewInterceptor accepting *slog.Logger
   → Returns a connect.Interceptor (unary + stream) that logs procedure name, duration, and Connect status code; *slog.Logger is a required constructor arg (no global/default slog); unit-tested with a fake handler

2. [backend] Wire logging.NewInterceptor into cmd/server/main.go; propagate *slog.Logger to all pkg constructors that log
   → *slog.Logger created once in main and passed explicitly to logging.NewInterceptor and any other pkg that currently calls slog.*; go test -race -count=1 ./... && go vet ./... passes
