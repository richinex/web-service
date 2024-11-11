// internal/config/config.go

package config

import (
    "fmt"
)

type Config struct {
    DatabaseURL string
    JWTSecret   string
    Environment string
}

func Load(getenv func(string) string) (*Config, error) {
    cfg := &Config{
        DatabaseURL: getenv("DATABASE_URL"),
        JWTSecret:   getenv("JWT_SECRET"),
        Environment: getenv("ENVIRONMENT"),
    }

    // Only JWT_SECRET is required for now since we're using in-memory store
    if cfg.JWTSecret == "" {
        return nil, fmt.Errorf("JWT_SECRET is required")
    }

    // Set defaults
    if cfg.Environment == "" {
        cfg.Environment = "development"
    }

    // If no DATABASE_URL, use in-memory
    if cfg.DatabaseURL == "" {
        cfg.DatabaseURL = "memory://"
    }

    return cfg, nil
}