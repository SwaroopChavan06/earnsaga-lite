package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DatabaseURL       string
	RedisURL          string
	JWTSecret         string
	GoogleClientID    string
	PubScaleAppID     string
	PubScalePubKey    string
	PubScaleSecretKey string // used for S2S callback signature verification
	Env               string
	AllowedOrigins    []string // CORS — comma-separated ALLOWED_ORIGINS env var
}

func Load() *Config {
	// Try loading .env from the current directory first, then the parent
	// (useful when running `go run ./cmd/server` from backend/ while .env
	// lives at the repo root). Production never has a .env file — Railway
	// and Docker inject real env vars — so missing files are not an error.
	if err := godotenv.Load(); err != nil {
		if err2 := godotenv.Load("../.env"); err2 != nil {
			log.Println("no .env file found, relying on system environment variables")
		}
	}

	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       mustEnv("DATABASE_URL"),
		RedisURL:          getEnv("REDIS_URL", ""),
		JWTSecret:         mustEnv("JWT_SECRET"),
		GoogleClientID:    mustEnv("GOOGLE_CLIENT_ID"),
		PubScaleAppID:     getEnv("PUBSCALE_APP_ID", "38002658"),
		PubScalePubKey:    getEnv("PUBSCALE_PUB_KEY", "C423E0560E41A9EF42876CC684CB1F74"),
		PubScaleSecretKey: mustEnv("PUBSCALE_SECRET_KEY"),
		Env:               getEnv("ENV", "development"),
		AllowedOrigins:    getEnvList("ALLOWED_ORIGINS", []string{"http://localhost:5173", "http://localhost:3000"}),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvList reads a comma-separated env var into a trimmed string slice.
// Falls back to the given defaults (typical local Vite/CRA dev origins) so
// local dev keeps working without ALLOWED_ORIGINS set — production deploys
// should always set it explicitly to the real frontend origin(s).
func getEnvList(key string, fallback []string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}
