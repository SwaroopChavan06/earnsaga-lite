package main

import (
	"log"
	"net/http"
	"time"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/config"
	"earnsaga-lite/internal/db"
	"earnsaga-lite/internal/handlers"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.Load()

	pool, err := db.NewPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	authHandler := &handlers.AuthHandler{DB: pool, Cfg: cfg}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		// Tighten this to your actual deployed frontend origin before submission.
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// --- Public routes ---
	r.Post("/auth/google", authHandler.GoogleLogin)

	// --- Protected routes (require JWT) ---
	r.Group(func(pr chi.Router) {
		pr.Use(auth.Middleware(cfg.JWTSecret))

		pr.Get("/me", authHandler.Me)

		// TODO next: offers list/search/detail, start-offer, wallet, leaderboard,
		// analytics event ingestion — wired the same way, using auth.UserIDFromContext(r.Context())
		// inside each handler to identify the caller.
	})

	// --- Callback route (PubScale S2S, no JWT — verified via signature instead) ---
	// TODO next: r.Get("/callbacks/pubscale", callbackHandler.Handle)

	log.Printf("earnsaga-lite server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
