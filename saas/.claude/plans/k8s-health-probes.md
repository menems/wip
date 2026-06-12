# k8s-health-probes
> Add `/healthz/live` and `/healthz/ready` HTTP endpoints so Kubernetes can manage pod lifecycle

**Created**: 2026-03-12 | **Branch**: feat/k8s-health-probes

## Steps
1. [backend] Create `pkg/health` package with liveness and readiness handlers
   → `GET /healthz/live` always returns 200; `GET /healthz/ready` calls injected `Checker` interface(s) and returns 200 or 503; `Handler` implements `RouteRegistrar`; unit-tested with a mock checker

2. [backend] Wire the health handler into `cmd/server/main.go`
   → Both probe endpoints are live on the running server; postgres pool is registered as the readiness checker via `pool.Ping`
