# slog-json-run-func
> Configure a structured JSON slog handler and consolidate os.Exit to a single call via a run() function

**Created**: 2026-03-12 | **Branch**: feat/slog-json-run-func

## Steps
1. [backend] Configure slog with a JSON handler
   → Early in startup, `slog.SetDefault` is called with a `slog.NewJSONHandler(os.Stderr, nil)` so every log line emits structured JSON; no other code changes needed

2. [backend] Extract `run()` function; single `os.Exit` in `main`
   → A `run(ctx context.Context) error` function contains all startup logic and returns errors instead of calling `os.Exit`; `mustEnv` is replaced by an inline error return; `main()` calls `run()` and invokes `os.Exit(1)` exactly once on non-nil error
