# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-02-27

### Added

- `pkg/problem`: RFC 7807 `application/problem+json` error response package with `request_id` correlation field ([#7])
- `pkg/telemetry`: OpenTelemetry `TracerProvider` supporting stdout exporter (dev) and OTLP exporter driven by `OTEL_EXPORTER_OTLP_ENDPOINT` (prod) ([#12])
- `RequestID` middleware — generates and propagates an `X-Request-Id` header per request ([#9])
- Structured JSON request logging middleware via `log/slog` with `method`, `path`, `status`, `duration_ms`, and `request_id` fields ([#9])
- Prometheus metrics middleware exposing `http_requests_total`, `http_request_duration_seconds`, and `http_requests_in_flight` series ([#10])
- `GET /metrics` endpoint for Prometheus scraping (internal use only) ([#10])
- `GET /health/live` liveness probe — returns `200 OK` while the process is running ([#11])
- `GET /health/ready` readiness probe with a pluggable `Checker` interface — returns `503 application/problem+json` if any checker fails ([#11])
- Multi-stage distroless `Dockerfile` producing minimal, non-root production images ([#13])
- GitHub Actions CI pipeline: `golangci-lint` + `govulncheck`, race-detection tests, binary build ([#14])
- `docs/adr/001-middleware-order.md` — ADR-001 documenting middleware chain rationale (`RequestID → Logger → Metrics → Recoverer`) ([#15])
- `docs/openapi.yaml` — OpenAPI 3.1 specification for all implemented routes ([#16])
- `docs/runbook.md` — ops runbook covering metrics, logs, tracing, and health probes ([#17])

### Changed

- Router migrated from `net/http.ServeMux` to `go-chi/chi/v5` for composable middleware chaining ([#6])
- Go version bumped to 1.25.7 to resolve known stdlib CVEs ([#8])
- `.golangci.yml` upgraded to golangci-lint v2 schema ([#8])

[Unreleased]: https://github.com/menems/sass/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/menems/sass/releases/tag/v0.1.0

[#6]: https://github.com/menems/sass/issues/6
[#7]: https://github.com/menems/sass/issues/7
[#8]: https://github.com/menems/sass/issues/8
[#9]: https://github.com/menems/sass/issues/9
[#10]: https://github.com/menems/sass/issues/10
[#11]: https://github.com/menems/sass/issues/11
[#12]: https://github.com/menems/sass/issues/12
[#13]: https://github.com/menems/sass/issues/13
[#14]: https://github.com/menems/sass/issues/14
[#15]: https://github.com/menems/sass/issues/15
[#16]: https://github.com/menems/sass/issues/16
[#17]: https://github.com/menems/sass/issues/17
