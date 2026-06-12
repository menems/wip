# pkg-server
> Extract HTTP server construction into a reusable pkg/server package

**Created**: 2026-03-12 | **Branch**: feat/pkg-server

## Steps
1. [backend] Create pkg/server with Server struct and functional options
   → Package exposes RouteRegistrar interface, Server struct, New(registrars, opts…) constructor, functional options WithAddr/WithReadHeaderTimeout, and Run(ctx) for listen + graceful shutdown; unit test covers happy-path start/stop via context cancellation

2. [backend] Wire cmd/server/main.go to pkg/server
   → main.go drops its local RouteRegistrar definition, addr() helper, raw http.Server block, and shutdown goroutine; replaces them with pkg/server.New(registrars, opts…).Run(ctx); QA gate (go test -race -count=1 ./… && go vet ./…) passes green
