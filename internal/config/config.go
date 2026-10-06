package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr, DatabaseURL, JWTSecret string
	JWTExpiry                    time.Duration
	AllowedOrigins               []string
}

func Load() (Config, error) {
	c := Config{Addr: env("APP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), JWTSecret: os.Getenv("JWT_SECRET")}
	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL is required")
	}
	if len(c.JWTSecret) < 32 {
		return c, errors.New("JWT_SECRET must be at least 32 characters")
	}
	d, err := time.ParseDuration(env("JWT_EXPIRY", "30m"))
	if err != nil || d < 5*time.Minute || d > 24*time.Hour {
		return c, errors.New("JWT_EXPIRY must be between 5m and 24h")
	}
	c.JWTExpiry = d
	for _, o := range strings.Split(env("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			c.AllowedOrigins = append(c.AllowedOrigins, o)
		}
	}
	return c, nil
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
