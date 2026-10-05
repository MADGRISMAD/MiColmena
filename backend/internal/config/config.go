// Package config carga la configuración desde variables de entorno.
package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr          string
	DatabaseURL   string
	JWTSecret     []byte
	TokenTTL      time.Duration
	CORSOrigins   []string
	AdminEmail    string
	AdminPassword string
}

func Load() (Config, error) {
	cfg := Config{
		Addr:          env("ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		JWTSecret:     []byte(os.Getenv("JWT_SECRET")),
		CORSOrigins:   splitList(env("CORS_ORIGINS", "http://localhost:5173")),
		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
	}

	ttl, err := time.ParseDuration(env("TOKEN_TTL", "24h"))
	if err != nil {
		return cfg, errors.New("TOKEN_TTL inválido: " + err.Error())
	}
	cfg.TokenTTL = ttl

	if cfg.DatabaseURL == "" {
		return cfg, errors.New("falta DATABASE_URL")
	}
	if len(cfg.JWTSecret) < 32 {
		return cfg, errors.New("JWT_SECRET debe tener al menos 32 caracteres")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
