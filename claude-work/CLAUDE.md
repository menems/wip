# Project context

## Module & toolchain
- Module: `github.com/blaz/serve` · Go **1.26**

## Makefile
```
make build   # go build ./cmd/serve/
make run     # go run ./cmd/serve/
make test    # go test ./...
make lint    # go vet ./...
make tidy    # go mod tidy
```

## Routing — http.ServeMux + ConnectRPC
```go
// platform/server/server.go — built-in routes
mux.HandleFunc("GET /health", s.healthHandler())
mux.HandleFunc("POST /echo", s.echoHandler())
mux.Handle("/", s.notFoundHandler())

// internal/{pkg}/connect_handler.go — feature routes
func (h *ConnectHandler) RegisterRoutes(mux *http.ServeMux) {
    path, handler := pkgv1connect.NewFooServiceHandler(h)
    mux.Handle(path, handler)
}
```

## Wiring a new feature
```go
// cmd/serve/main.go
repo := pkg.NewRepository()
svc  := pkg.NewService(repo)
log  := slog.New(slog.NewJSONHandler(os.Stdout, nil))
srv  := server.New(cfg, log, server.WithRoutes(pkg.NewConnectHandler(svc)))
```

## Test helpers
```go
// internal/{pkg}/connect_handler_test.go
func newConnectServer() *httptest.Server {
    repo := pkg.NewRepository()
    svc  := pkg.NewService(repo)
    h    := pkg.NewConnectHandler(svc)
    mux  := http.NewServeMux()
    h.RegisterRoutes(mux)
    return httptest.NewServer(mux)
}

// platform/server/handlers_test.go
func newTestServer() *server.Server {
    return server.New(server.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}
```
