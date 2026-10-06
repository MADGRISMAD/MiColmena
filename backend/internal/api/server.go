// Package api expone la API HTTP de MiColmena.
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
	"github.com/MADGRISMAD/MiColmena/backend/internal/ratelimit"
)

// Options configura el servidor. Los valores en cero usan los valores por defecto.
type Options struct {
	// CORSOrigins son los orígenes del frontend permitidos ("*" permite cualquiera).
	CORSOrigins []string
	// TrustProxy indica que la API corre detrás de un proxy de confianza (Caddy, Nginx…)
	// que añade la IP real del cliente al final de X-Forwarded-For. Con false esa cabecera
	// se ignora, porque cualquiera podría falsificarla para esquivar el límite de intentos.
	TrustProxy bool
	// AuthRatePerMin son los intentos por minuto y por IP en /api/auth/*. Por defecto 10.
	AuthRatePerMin int
	// LoginMaxFailures son los fallos seguidos de un mismo email que lo bloquean. Por defecto 5.
	LoginMaxFailures int
	// LoginLockout es lo que dura ese bloqueo. Por defecto 15 minutos.
	LoginLockout time.Duration
}

type Server struct {
	db          *pgxpool.Pool
	tokens      *auth.Issuer
	corsOrigins []string
	trustProxy  bool
	authLimit   *ratelimit.Limiter
	loginLock   *ratelimit.Lockout
}

func NewServer(db *pgxpool.Pool, tokens *auth.Issuer, opts Options) *Server {
	if opts.AuthRatePerMin == 0 {
		opts.AuthRatePerMin = 10
	}
	if opts.LoginMaxFailures == 0 {
		opts.LoginMaxFailures = 5
	}
	if opts.LoginLockout == 0 {
		opts.LoginLockout = 15 * time.Minute
	}
	return &Server{
		db:          db,
		tokens:      tokens,
		corsOrigins: opts.CORSOrigins,
		trustProxy:  opts.TrustProxy,
		authLimit:   ratelimit.New(opts.AuthRatePerMin),
		loginLock:   ratelimit.NewLockout(opts.LoginMaxFailures, opts.LoginLockout),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.health)

	mux.Handle("POST /api/auth/register", s.limitAuth(s.register))
	mux.Handle("POST /api/auth/login", s.limitAuth(s.login))
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

// limitAuth limita los intentos por IP en las rutas de registro y login.
func (s *Server) limitAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, retry := s.authLimit.Allow(clientIP(r, s.trustProxy))
		if !ok {
			tooManyRequests(w, retry)
			return
		}
		next(w, r)
	})
}

// tooManyRequests responde 429 con Retry-After (en segundos, redondeado hacia arriba).
func tooManyRequests(w http.ResponseWriter, retry time.Duration) {
	seconds := int((retry + time.Second - 1) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeError(w, http.StatusTooManyRequests,
		fmt.Sprintf("demasiados intentos, inténtalo de nuevo en %s", waitText(seconds)))
}

func waitText(seconds int) string {
	if seconds == 1 {
		return "1 segundo"
	}
	if seconds < 90 {
		return fmt.Sprintf("%d segundos", seconds)
	}
	return fmt.Sprintf("%d minutos", (seconds+59)/60)
}

// clientIP devuelve la IP del cliente usada como clave del límite. Las IPv6 se agrupan por /64,
// porque una sola persona suele tener un bloque entero y podría rotar de dirección dentro de él.
func clientIP(r *http.Request, trustProxy bool) string {
	raw := ""
	if trustProxy {
		// El proxy de confianza añade la IP real al final; lo que venga antes lo pone el cliente.
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			raw = strings.TrimSpace(parts[len(parts)-1])
		}
	}
	if net.ParseIP(raw) == nil {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		raw = host
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return raw
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

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
