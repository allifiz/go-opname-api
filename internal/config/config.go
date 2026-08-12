package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort        string
	DatabaseURL    string
	JWTSecret      string
	BootstrapToken string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		AppPort:        getEnv("APP_PORT", "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		BootstrapToken: os.Getenv("BOOTSTRAP_TOKEN"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if cfg.BootstrapToken != "" && len(cfg.BootstrapToken) < 32 {
		return Config{}, fmt.Errorf("BOOTSTRAP_TOKEN must be at least 32 characters when configured")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
