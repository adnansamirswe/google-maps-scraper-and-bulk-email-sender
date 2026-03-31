package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration loaded from .env.
type Config struct {
	Password  string
	JWTSecret string
	Port      string
	DataDir   string
}

// Load reads the .env file and returns a Config.
func Load() (*Config, error) {
	// Load .env if it exists (ignore error if missing)
	_ = godotenv.Load()

	cfg := &Config{
		Password:  getEnv("APP_PASSWORD", "admin123"),
		JWTSecret: getEnv("JWT_SECRET", "default-secret-change-me"),
		Port:      getEnv("PORT", "3001"),
		DataDir:   getEnv("DATA_DIR", "./data"),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return fallback
}
