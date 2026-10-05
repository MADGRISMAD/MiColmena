// Comando api arranca el servidor HTTP de MiColmena.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
	"github.com/MADGRISMAD/MiColmena/backend/internal/config"
	"github.com/MADGRISMAD/MiColmena/backend/internal/db"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("el servidor terminó con error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	if err := ensureAdmin(ctx, pool, cfg); err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.NewServer(pool, auth.NewIssuer(cfg.JWTSecret, cfg.TokenTTL), cfg.CORSOrigins).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("servidor escuchando", "addr", cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("apagando servidor")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
	return nil
}

// ensureAdmin crea el primer administrador desde ADMIN_EMAIL y ADMIN_PASSWORD si aún no existe ninguno.
func ensureAdmin(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) error {
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		return nil
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE role = 'admin')`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	if len(cfg.AdminPassword) < 8 {
		return errors.New("ADMIN_PASSWORD debe tener al menos 8 caracteres")
	}
	hash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		return err
	}
	// Si el email ya lo registró otra persona no se le da el rol de administrador:
	// eso permitiría quedarse con la cuenta registrándose antes que el dueño.
	tag, err := pool.Exec(ctx, `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ('Administrador', $1, $2, 'admin')
		ON CONFLICT (lower(email)) DO NOTHING`,
		cfg.AdminEmail, hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		slog.Warn("ADMIN_EMAIL ya está registrado como otro usuario; no se creó el administrador", "email", cfg.AdminEmail)
		return nil
	}
	slog.Info("administrador inicial creado", "email", cfg.AdminEmail)
	return nil
}
