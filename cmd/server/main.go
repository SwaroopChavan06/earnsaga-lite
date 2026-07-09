package main

import (
	"log"
	"net/http"
	"time"

	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/config"
	"earnsaga-lite/internal/db"
	"earnsaga-lite/internal/handlers"
	"earnsaga-lite/internal/pubscale"

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
	userHandler := &handlers.UserHandler{DB: pool, Cfg: cfg}
	psClient := pubscale.NewClient(cfg.PubScaleAppID, cfg.PubScalePubKey)
	offersHandler := &handlers.OffersHandler{DB: pool, Cfg: cfg, PubScale: psClient}
	offerActionsHandler := &handlers.OfferActionsHandler{DB: pool}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	standardTimeout := chimw.Timeout(30 * time.Second)

	r.With(standardTimeout).Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// --- API v1 Routes ---
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes
		r.With(standardTimeout).Post("/auth/google", authHandler.GoogleLogin)

		if cfg.Env == "development" {
			devHandler := &handlers.DevHandler{DB: pool, Cfg: cfg}
			r.With(standardTimeout).Get("/dev/token", devHandler.IssueDevToken)
		}

		// Protected routes
		r.Group(func(pr chi.Router) {
			pr.Use(auth.Middleware(cfg.JWTSecret))
			pr.Use(standardTimeout)

			// User domain
			pr.Get("/users/profile", userHandler.GetProfile)
			pr.Get("/users/wallet", userHandler.GetWallet)
			pr.Get("/users/transactions", userHandler.GetTransactions)

			// Offer domain
			pr.Get("/offers", offersHandler.ListOffers)
			pr.Get("/offers/{id}", offersHandler.GetOfferDetail)
			pr.Post("/offers/{id}/start", offerActionsHandler.StartOffer)
		})

		// Admin routes
		r.Group(func(ar chi.Router) {
			ar.Use(auth.Middleware(cfg.JWTSecret))
			ar.Use(handlers.RequireAdmin(pool))
			ar.Post("/admin/sync-offers", offersHandler.SyncOffers)
		})
	})

	log.Printf("earnsaga-lite server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
