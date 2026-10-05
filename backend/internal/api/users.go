package api

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
)

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

const userColumns = `id, name, email, role, created_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt)
	return u, err
}

type authResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.TrimSpace(in.Email)

	errs := validationErrors{}
	errs.check(notBlank(in.Name), "name", "es obligatorio")
	_, mailErr := mail.ParseAddress(in.Email)
	errs.check(mailErr == nil, "email", "no es un email válido")
	errs.check(len(in.Password) >= 8, "password", "debe tener al menos 8 caracteres")
	errs.check(len(in.Password) <= 72, "password", "debe tener como máximo 72 caracteres")
	if errs.write(w) {
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		internalError(w, err)
		return
	}

	// El registro público siempre crea clientes; los agentes los asigna un administrador.
	u, err := scanUser(s.db.QueryRow(r.Context(), `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, 'customer')
		RETURNING `+userColumns,
		in.Name, in.Email, hash))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "ese email ya está registrado")
			return
		}
		internalError(w, err)
		return
	}

	s.respondWithToken(w, http.StatusCreated, u)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}

	var u User
	var hash string
	err := s.db.QueryRow(r.Context(), `
		SELECT `+userColumns+`, password_hash FROM users WHERE lower(email) = lower($1)`,
		strings.TrimSpace(in.Email),
	).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &hash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		internalError(w, err)
		return
	}
	if err != nil || !auth.CheckPassword(hash, in.Password) {
		writeError(w, http.StatusUnauthorized, "email o contraseña incorrectos")
		return
	}

	s.respondWithToken(w, http.StatusOK, u)
}

func (s *Server) respondWithToken(w http.ResponseWriter, status int, u User) {
	token, expires, err := s.tokens.Issue(u.ID, u.Role)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, status, authResponse{Token: token, ExpiresAt: expires, User: u})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, err := scanUser(s.db.QueryRow(r.Context(),
		`SELECT `+userColumns+` FROM users WHERE id = $1`, claimsFrom(r).UserID()))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnauthorized, "el usuario ya no existe")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// listUsers devuelve usuarios, opcionalmente filtrados por rol (?role=agent).
func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	if role != "" && !validRole(role) {
		writeError(w, http.StatusBadRequest, "rol inválido")
		return
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT `+userColumns+` FROM users
		WHERE ($1 = '' OR role = $1)
		ORDER BY name
		LIMIT 500`, role)
	if err != nil {
		internalError(w, err)
		return
	}
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (User, error) { return scanUser(row) })
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": users})
}

func (s *Server) updateUserRole(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validRole(in.Role) {
		writeError(w, http.StatusUnprocessableEntity, "rol inválido")
		return
	}
	if id == claimsFrom(r).UserID() && in.Role != auth.RoleAdmin {
		writeError(w, http.StatusUnprocessableEntity, "no puedes quitarte el rol de administrador")
		return
	}

	u, err := scanUser(s.db.QueryRow(r.Context(),
		`UPDATE users SET role = $2 WHERE id = $1 RETURNING `+userColumns, id, in.Role))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func validRole(role string) bool {
	return role == auth.RoleAdmin || role == auth.RoleAgent || role == auth.RoleCustomer
}
