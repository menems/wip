# ctx-logger
> inject *slog.Logger into request context so handlers can retrieve it via logging.FromContext

**Created**: 2026-03-12 | **Branch**: feat/ctx-logger

## Steps
1. [backend] Add `WithLogger` / `FromContext` context helpers in `pkg/logging`
   → `logging.WithLogger(ctx, log)` stores a `*slog.Logger` in context; `logging.FromContext(ctx)` retrieves it, falling back to `slog.Default()`; unit tests cover round-trip and the fallback path

2. [backend] Inject logger into context before calling `next` in `Interceptor.WrapUnary` and `WrapStreamingHandler`
   → Interceptor calls `logging.WithLogger(ctx, i.log)` before forwarding the request; a handler stub in the interceptor test verifies `logging.FromContext(ctx)` returns the injected logger (not the default)
