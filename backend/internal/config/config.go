package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port           string
	MongoURI       string
	MongoDB        string
	// RedisURL is the full connection URL (e.g. rediss://... from Upstash).
	// If set it takes priority over RedisAddr + RedisPassword.
	RedisURL       string
	RedisAddr      string
	RedisPassword  string
	JWTSecret      string
	JWTExpiryHours int
	AllowedOrigins string
}

// Load reads the .env file (if present) and populates Config from environment variables.
func Load() (*Config, error) {
	// .env is optional; in production env vars are injected directly.
	_ = godotenv.Load()

	expiry, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "72"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: %w", err)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET must be set")
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		MongoURI:       getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDB:        getEnv("MONGODB_DB", "pollster"),
		RedisURL:       getEnv("REDIS_URL", ""),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		JWTSecret:      secret,
		JWTExpiryHours: expiry,
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
