package config

import (
	"log"
	"os"

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
}

func Load() *Config {
	// .env is optional in production (Railway injects real env vars),
	// so we don't fail if it's missing — just log and move on.
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on system environment variables")
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
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}
