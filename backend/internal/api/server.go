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
	// UploadDir es la carpeta donde se guardan los adjuntos. Por defecto "data/uploads".
	UploadDir string
	// MaxUploadBytes es el tamaño máximo de cada adjunto. Por defecto 10 MB.
	MaxUploadBytes int64
	// AppURL es la dirección del frontend, para los enlaces de los correos. Por defecto http://localhost:5173.
	AppURL string
}

type Server struct {
	db          *pgxpool.Pool
	tokens      *auth.Issuer
	corsOrigins []string
	trustProxy  bool
	authLimit   *ratelimit.Limiter
	loginLock   *ratelimit.Lockout
	uploadDir   string
	maxUpload   int64
	appURL      string
	broker      *broker
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
	if opts.UploadDir == "" {
		opts.UploadDir = "data/uploads"
	}
	if opts.MaxUploadBytes == 0 {
		opts.MaxUploadBytes = 10 << 20
	}
	if opts.AppURL == "" {
		opts.AppURL = "http://localhost:5173"
	}
	return &Server{
		db:          db,
		tokens:      tokens,
		corsOrigins: opts.CORSOrigins,
		trustProxy:  opts.TrustProxy,
		authLimit:   ratelimit.New(opts.AuthRatePerMin),
		loginLock:   ratelimit.NewLockout(opts.LoginMaxFailures, opts.LoginLockout),
		uploadDir:   opts.UploadDir,
		maxUpload:   opts.MaxUploadBytes,
		appURL:      strings.TrimSuffix(opts.AppURL, "/"),
		broker:      newBroker(),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.health)

	mux.Handle("POST /api/auth/register", s.limitAuth(s.register))
	mux.Handle("POST /api/auth/login", s.limitAuth(s.login))
	mux.Handle("POST /api/auth/forgot", s.limitAuth(s.forgotPassword))
	mux.Handle("POST /api/auth/reset", s.limitAuth(s.resetPassword))
	mux.Handle("GET /api/me", s.authed(s.me))
	mux.Handle("PATCH /api/me", s.authed(s.updateMe))
	mux.Handle("POST /api/me/password", s.authed(s.changePassword))
	mux.Handle("GET /api/notifications", s.authed(s.listNotifications))
	mux.Handle("POST /api/notifications/read", s.authed(s.readNotifications))
	mux.Handle("GET /api/stream", s.authed(s.stream))

	mux.Handle("GET /api/tickets", s.authed(s.listTickets))
	mux.Handle("POST /api/tickets", s.authed(s.createTicket))
	mux.Handle("GET /api/tickets/{id}", s.authed(s.getTicket))
	mux.Handle("PATCH /api/tickets/{id}", s.authed(s.updateTicket))
	mux.Handle("POST /api/tickets/bulk", s.staffOnly(s.bulkUpdateTickets))
	mux.Handle("GET /api/tickets/{id}/events", s.authed(s.listEvents))
	mux.Handle("POST /api/tickets/{id}/satisfaction", s.authed(s.rateTicket))
	mux.Handle("GET /api/tickets/{id}/attachments", s.authed(s.listAttachments))
	mux.Handle("POST /api/tickets/{id}/attachments", s.authed(s.uploadAttachment))
	mux.Handle("GET /api/attachments/{id}", s.authed(s.downloadAttachment))
	mux.Handle("DELETE /api/attachments/{id}", s.authed(s.deleteAttachment))
	mux.Handle("GET /api/tickets/{id}/comments", s.authed(s.listComments))
	mux.Handle("POST /api/tickets/{id}/comments", s.authed(s.createComment))

	mux.Handle("GET /api/users", s.staffOnly(s.listUsers))
	mux.Handle("POST /api/users", s.adminOnly(s.createUser))
	mux.Handle("PATCH /api/users/{id}", s.adminOnly(s.updateUser))
	mux.Handle("PATCH /api/users/{id}/role", s.adminOnly(s.updateUserRole))

	mux.Handle("GET /api/tags", s.staffOnly(s.listTags))
	mux.Handle("GET /api/sla", s.staffOnly(s.listSLA))
	mux.Handle("GET /api/reports", s.staffOnly(s.report))
	mux.Handle("GET /api/reports/tickets.csv", s.staffOnly(s.exportTickets))
	mux.Handle("GET /api/views", s.authed(s.listViews))
	mux.Handle("POST /api/views", s.authed(s.createView))
	mux.Handle("DELETE /api/views/{id}", s.authed(s.deleteView))

	// La base de conocimiento publicada se puede leer sin iniciar sesión.
	mux.Handle("GET /api/articles", s.optionalAuth(s.listArticles))
	mux.Handle("GET /api/articles/{id}", s.optionalAuth(s.getArticle))
	mux.Handle("POST /api/articles", s.staffOnly(s.createArticle))
	mux.Handle("PATCH /api/articles/{id}", s.staffOnly(s.updateArticle))
	mux.Handle("DELETE /api/articles/{id}", s.staffOnly(s.deleteArticle))
	mux.Handle("PUT /api/sla", s.adminOnly(s.updateSLA))
	mux.Handle("GET /api/categories", s.authed(s.listCategories))
	mux.Handle("POST /api/categories", s.adminOnly(s.createCategory))
	mux.Handle("PATCH /api/categories/{id}", s.adminOnly(s.renameCategory))
	mux.Handle("DELETE /api/categories/{id}", s.adminOnly(s.deleteCategory))
	mux.Handle("GET /api/macros", s.staffOnly(s.listMacros))
	mux.Handle("POST /api/macros", s.staffOnly(s.createMacro))
	mux.Handle("PATCH /api/macros/{id}", s.staffOnly(s.updateMacro))
	mux.Handle("DELETE /api/macros/{id}", s.staffOnly(s.deleteMacro))
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
		claims, status, msg := s.authenticate(r)
		if status != 0 {
			if status == http.StatusInternalServerError {
				writeError(w, status, "error interno")
				return
			}
			writeError(w, status, msg)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims)))
	})
}

// optionalAuth deja pasar sin sesión (claims vacíos); si hay un token válido, lo usa.
func (s *Server) optionalAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claims, status, _ := s.authenticate(r); status == 0 {
			r = r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims))
		}
		next(w, r)
	})
}

// authenticate valida el token Bearer. Devuelve un código de error distinto de 0 si no es válido.
func (s *Server) authenticate(r *http.Request) (auth.Claims, int, string) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" {
		return auth.Claims{}, http.StatusUnauthorized, "falta el token de acceso"
	}
	claims, err := s.tokens.Parse(token)
	if err != nil {
		return auth.Claims{}, http.StatusUnauthorized, "token inválido o expirado"
	}
	// El rol se lee de la base de datos (búsqueda por clave primaria) para que
	// un cambio de rol o un usuario eliminado se apliquen sin esperar a que expire el token.
	var active bool
	var validAfter time.Time
	if err := s.db.QueryRow(r.Context(),
		`SELECT role, active, tokens_valid_after FROM users WHERE id = $1`, claims.UserID(),
	).Scan(&claims.Role, &active, &validAfter); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.Claims{}, http.StatusUnauthorized, "el usuario ya no existe"
		}
		slog.Error("error interno", "error", err)
		return auth.Claims{}, http.StatusInternalServerError, ""
	}
	if !active {
		return auth.Claims{}, http.StatusUnauthorized, "esta cuenta está desactivada"
	}
	// Un cambio de contraseña invalida los tokens emitidos antes.
	if claims.IssuedAt == nil || claims.IssuedAt.Before(validAfter) {
		return auth.Claims{}, http.StatusUnauthorized, "la sesión ya no es válida, vuelve a iniciar sesión"
	}
	return claims, 0, ""
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
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

// Unwrap deja que http.ResponseController llegue a la conexión (Flush, plazos de lectura y escritura).
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

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
