// Package api expone la API HTTP de MiColmena.
package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
)

type Server struct {
	db          *pgxpool.Pool
	tokens      *auth.Issuer
	corsOrigins []string
}

func NewServer(db *pgxpool.Pool, tokens *auth.Issuer, corsOrigins []string) *Server {
	return &Server{db: db, tokens: tokens, corsOrigins: corsOrigins}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.health)

	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.Handle("GET /api/me", s.authed(s.me))

	mux.Handle("GET /api/tickets", s.authed(s.listTickets))
	mux.Handle("POST /api/tickets", s.authed(s.createTicket))
	mux.Handle("GET /api/tickets/{id}", s.authed(s.getTicket))
	mux.Handle("PATCH /api/tickets/{id}", s.authed(s.updateTicket))
	mux.Handle("GET /api/tickets/{id}/comments", s.authed(s.listComments))
	mux.Handle("POST /api/tickets/{id}/comments", s.authed(s.createComment))

	mux.Handle("GET /api/users", s.staffOnly(s.listUsers))
	mux.Handle("PATCH /api/users/{id}/role", s.adminOnly(s.updateUserRole))
	mux.Handle("GET /api/stats", s.staffOnly(s.stats))

	return s.recoverer(s.logger(s.cors(mux)))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "base de datos no disponible")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Middleware ---

type claimsKey struct{}

func claimsFrom(r *http.Request) auth.Claims {
	c, _ := r.Context().Value(claimsKey{}).(auth.Claims)
	return c
}

func (s *Server) authed(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "falta el token de acceso")
			return
		}
		claims, err := s.tokens.Parse(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "token inválido o expirado")
			return
		}
		// El rol se lee de la base de datos (búsqueda por clave primaria) para que
		// un cambio de rol o un usuario eliminado se apliquen sin esperar a que expire el token.
		if err := s.db.QueryRow(r.Context(),
			`SELECT role FROM users WHERE id = $1`, claims.UserID(),
		).Scan(&claims.Role); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusUnauthorized, "el usuario ya no existe")
				return
			}
			internalError(w, err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims)))
	})
}

func (s *Server) staffOnly(next http.HandlerFunc) http.Handler {
	return s.authed(func(w http.ResponseWriter, r *http.Request) {
		if !claimsFrom(r).IsStaff() {
			writeError(w, http.StatusForbidden, "solo para agentes y administradores")
			return
		}
		next(w, r)
	})
}

func (s *Server) adminOnly(next http.HandlerFunc) http.Handler {
	return s.authed(func(w http.ResponseWriter, r *http.Request) {
		if claimsFrom(r).Role != auth.RoleAdmin {
			writeError(w, http.StatusForbidden, "solo para administradores")
			return
		}
		next(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (slices.Contains(s.corsOrigins, "*") || slices.Contains(s.corsOrigins, origin)) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *Server) logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("petición",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start).Round(time.Microsecond),
		)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic", "error", err, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "error interno")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
