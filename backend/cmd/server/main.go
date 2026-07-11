package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"earnsaga-lite/internal/analytics"
	"earnsaga-lite/internal/auth"
	"earnsaga-lite/internal/cache"
	"earnsaga-lite/internal/callback"
	"earnsaga-lite/internal/config"
	"earnsaga-lite/internal/db"
	"earnsaga-lite/internal/leaderboard"
	"earnsaga-lite/internal/offer"
	"earnsaga-lite/internal/pubscale"
	"earnsaga-lite/internal/user"
	"earnsaga-lite/internal/wallet"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.Load()

	log.Println("running database migrations...")
	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	pool, err := db.NewPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	redisClient, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}
	defer redisClient.Close()

	psClient := pubscale.NewClient(cfg.PubScaleAppID, cfg.PubScalePubKey)

	// Repositories
	userRepo := &user.Repository{DB: pool}
	walletRepo := &wallet.Repository{DB: pool}
	leaderboardRepo := &leaderboard.Repository{DB: pool, Redis: redisClient}
	analyticsRepo := &analytics.Repository{DB: pool}
	offerRepo := &offer.Repository{DB: pool}
	callbackRepo := &callback.Repository{DB: pool}

	// Services
	userService := &user.Service{Repo: userRepo}
	walletService := &wallet.Service{Repo: walletRepo}
	leaderboardService := &leaderboard.Service{Repo: leaderboardRepo}
	analyticsService := &analytics.Service{Repo: analyticsRepo}
	offerService := &offer.Service{Repo: offerRepo, PubScale: psClient}
	callbackService := &callback.Service{Repo: callbackRepo, SecretKey: cfg.PubScaleSecretKey, Leaderboard: leaderboardService}

	// Leaderboard broadcaster — one goroutine polls Redis every 3s and fans
	// out to all SSE clients. Started here so its lifetime is tied to the
	// server process, not any individual request.
	lbBroadcaster := leaderboard.NewBroadcaster(leaderboardService, 3*time.Second)
	go lbBroadcaster.Run(context.Background())

	// Handlers
	authHandler := &auth.Handler{UserService: userService, Cfg: cfg}
	userHandler := &user.Handler{Service: userService}
	walletHandler := &wallet.Handler{Service: walletService}
	leaderboardHandler := &leaderboard.Handler{Service: leaderboardService, Broadcaster: lbBroadcaster}
	analyticsHandler := &analytics.Handler{Service: analyticsService}
	offerHandler := &offer.Handler{Service: offerService}
	callbackHandler := &callback.Handler{Service: callbackService}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Applied per-route/group, not globally — long-running admin jobs
	// (offer sync) must not be cut off at 30s.
	standardTimeout := chimw.Timeout(30 * time.Second)

	r.With(standardTimeout).Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// PubScale S2S callback — public (no JWT), authenticated instead via
	// the MD5 signature check inside the handler itself.
	r.With(standardTimeout).Get("/callbacks/pubscale", callbackHandler.HandlePubScale)

	r.Route("/api/v1", func(r chi.Router) {
		// --- Public auth routes ---
		r.With(standardTimeout).Post("/auth/google", authHandler.GoogleLogin)
		if cfg.Env == "development" {
			r.With(standardTimeout).Get("/dev/token", authHandler.IssueDevToken)
		}

		// --- Protected routes ---
		r.Group(func(pr chi.Router) {
			pr.Use(auth.Middleware(cfg.JWTSecret))
			pr.Use(standardTimeout)

			// User domain
			pr.Get("/users/profile", userHandler.GetProfile)
			pr.Get("/users/wallet", walletHandler.GetWallet)
			pr.Get("/users/wallet/transactions", walletHandler.GetTransactions)
			// Collocated: the Wallet page always needs both balance and
			// transactions together, so this replaces two round-trips with
			// one (see wallet.Service.GetSummary for the parallel queries).
			pr.Get("/users/wallet/summary", walletHandler.GetWalletSummary)

			// Offers
			pr.Get("/offers", offerHandler.ListOffers)
			pr.Get("/offers/{id}", offerHandler.GetOfferDetail)
			pr.Post("/offers/{id}/start", offerHandler.StartOffer)

			// Leaderboard
			pr.Get("/leaderboard", leaderboardHandler.GetLeaderboard)

			// Events — batch is the preferred ingestion path for the
			// frontend's client-side event queue; single is kept for any
			// other caller that wants immediate, one-off tracking.
			pr.Post("/events", analyticsHandler.TrackEvent)
			pr.Post("/events/batch", analyticsHandler.TrackEventBatch)
		})

		// --- Real-time SSE routes — deliberately their own group WITHOUT
		// standardTimeout, since a long-lived stream connection would
		// otherwise get killed at 30s by chi's Timeout middleware. ---
		r.Group(func(sr chi.Router) {
			sr.Use(auth.Middleware(cfg.JWTSecret))
			sr.Get("/leaderboard/stream", leaderboardHandler.StreamLeaderboard)
		})

		// --- Admin routes — properly gated with RequireAdmin now, not just
		// "logged in". This was a real security gap before this pass. ---
		r.Group(func(ar chi.Router) {
			ar.Use(auth.Middleware(cfg.JWTSecret))
			ar.Use(auth.RequireAdmin(userService))
			ar.Get("/admin/analytics", analyticsHandler.GetAnalytics)

			// Sync deliberately has NO standardTimeout — it can run for
			// minutes syncing thousands of offers. It has its own internal
			// 5-minute cap (see offer.Handler.SyncOffers) so it can't hang forever.
			ar.Post("/admin/sync-offers", offerHandler.SyncOffers)
		})
	})

	log.Printf("earnsaga-lite server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
