package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"sassai/backend/internal/adapter/jwt"
	adapthttp "sassai/backend/internal/adapter/http"
	"sassai/backend/internal/auth"
	"sassai/backend/internal/db"
	"sassai/backend/internal/user"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	ctx := context.Background()

	// ── Infrastructure ───────────────────────────────────────────────────────
	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// ── Driven adapters (secondary / outbound) ───────────────────────────────
	userRepo := db.NewUserRepository(pool)
	tokenSvc := jwt.New()

	// ── Application services (use cases) ────────────────────────────────────
	userSvc := user.NewService(userRepo)
	authSvc := auth.NewService(userRepo, tokenSvc)

	// ── Driving adapters (primary / inbound) ─────────────────────────────────
	userHandler := adapthttp.NewUserHandler(userSvc)
	authHandler := adapthttp.NewAuthHandler(authSvc)

	// ── Router ───────────────────────────────────────────────────────────────
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	})

	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(adapthttp.RequireAuth(tokenSvc))
		r.Get("/api/users/me", userHandler.GetMe)
		r.Patch("/api/users/me", userHandler.UpdateMe)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server listening on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
