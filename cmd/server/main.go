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
	psClient := pubscale.NewClient(cfg.PubScaleAppID, cfg.PubScalePubKey)
	offersHandler := &handlers.OffersHandler{DB: pool, Cfg: cfg, PubScale: psClient}
	offerActionsHandler := &handlers.OfferActionsHandler{DB: pool}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		// Tighten this to your actual deployed frontend origin before submission.
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Applied per-route (not globally) so long-running admin jobs like
	// offer sync aren't cut off at 30s. r.With(...) scopes middleware to
	// only the next route registration, unlike r.Use(...) which is global.
	standardTimeout := chimw.Timeout(30 * time.Second)

	r.With(standardTimeout).Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// --- Public routes ---
	r.With(standardTimeout).Post("/auth/google", authHandler.GoogleLogin)

	if cfg.Env == "development" {
		devHandler := &handlers.DevHandler{DB: pool, Cfg: cfg}
		r.With(standardTimeout).Get("/dev/token", devHandler.IssueDevToken)
	}

	// --- Protected routes (require JWT) ---
	r.Group(func(pr chi.Router) {
		pr.Use(auth.Middleware(cfg.JWTSecret))
		pr.Use(standardTimeout) // fine to apply at group level here — nothing in this group is long-running except the admin subgroup below, which sits outside this Use call

		pr.Get("/users/profile", authHandler.GetProfile)

		pr.Get("/offers", offersHandler.ListOffers)
		pr.Get("/offers/{id}", offersHandler.GetOfferDetail)
		pr.Post("/offers/{id}/start", offerActionsHandler.StartOffer)

		// TODO next: wallet, leaderboard, analytics event ingestion.
	})

	// Admin routes get their own group WITHOUT standardTimeout, since sync
	// jobs can legitimately run for minutes. SyncOffers itself still has an
	// internal 5-minute cap via context.WithTimeout, so this can't hang forever.
	r.Group(func(ar chi.Router) {
		ar.Use(auth.Middleware(cfg.JWTSecret))
		ar.Use(handlers.RequireAdmin(pool))
		ar.Post("/admin/sync-offers", offersHandler.SyncOffers)
		// TODO next: ar.Get("/admin/analytics", analyticsHandler.Report)
	})

	// --- Callback route (PubScale S2S, no JWT — verified via signature instead) ---
	// TODO next: r.Get("/callbacks/pubscale", callbackHandler.Handle)

	log.Printf("earnsaga-lite server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
