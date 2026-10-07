// Package config carga la configuración desde variables de entorno.
package config

import (
	"errors"
	"os"
	"strconv"
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

	// TrustProxy: la API corre detrás de un proxy de confianza que añade la IP real al final
	// de X-Forwarded-For. Solo debe activarse si de verdad es así.
	TrustProxy       bool
	AuthRatePerMin   int
	LoginMaxFailures int
	LoginLockout     time.Duration

	// Adjuntos: carpeta donde se guardan y tamaño máximo de cada archivo.
	UploadDir      string
	MaxUploadBytes int64

	// AppURL es la dirección del frontend, para los enlaces de los correos.
	AppURL string
	// Correo saliente. Sin SMTPHost, los correos solo se escriben en el log.
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
}

func Load() (Config, error) {
	cfg := Config{
		Addr:          env("ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		JWTSecret:     []byte(os.Getenv("JWT_SECRET")),
		CORSOrigins:   splitList(env("CORS_ORIGINS", "http://localhost:5173")),
		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		UploadDir:     env("UPLOAD_DIR", "data/uploads"),
		AppURL:        strings.TrimSuffix(env("APP_URL", "http://localhost:5173"), "/"),
		SMTPHost:      os.Getenv("SMTP_HOST"),
		SMTPUsername:  os.Getenv("SMTP_USERNAME"),
		SMTPPassword:  os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:      env("SMTP_FROM", "BeHIve <no-reply@localhost>"),
	}

	ttl, err := time.ParseDuration(env("TOKEN_TTL", "24h"))
	if err != nil {
		return cfg, errors.New("TOKEN_TTL inválido: " + err.Error())
	}
	cfg.TokenTTL = ttl

	if cfg.TrustProxy, err = strconv.ParseBool(env("TRUST_PROXY", "false")); err != nil {
		return cfg, errors.New("TRUST_PROXY inválido: usa true o false")
	}
	if cfg.AuthRatePerMin, err = positiveInt("AUTH_RATE_PER_MIN", 10); err != nil {
		return cfg, err
	}
	if cfg.LoginMaxFailures, err = positiveInt("LOGIN_MAX_FAILURES", 5); err != nil {
		return cfg, err
	}
	if cfg.LoginLockout, err = time.ParseDuration(env("LOGIN_LOCKOUT", "15m")); err != nil || cfg.LoginLockout <= 0 {
		return cfg, errors.New("LOGIN_LOCKOUT inválido: usa una duración como 15m")
	}

	maxMB, err := positiveInt("MAX_UPLOAD_MB", 10)
	if err != nil {
		return cfg, err
	}
	cfg.MaxUploadBytes = int64(maxMB) << 20

	if cfg.SMTPPort, err = positiveInt("SMTP_PORT", 587); err != nil {
		return cfg, err
	}

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

func positiveInt(key string, fallback int) (int, error) {
	n, err := strconv.Atoi(env(key, strconv.Itoa(fallback)))
	if err != nil || n < 1 {
		return 0, errors.New(key + " inválido: debe ser un número entero mayor que 0")
	}
	return n, nil
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
