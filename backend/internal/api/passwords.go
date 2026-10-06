package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
)

const resetTokenTTL = time.Hour

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// forgotPassword envía un enlace para elegir una contraseña nueva. Responde lo mismo exista o no
// la cuenta, para no revelar qué emails están registrados.
func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	var userID int64
	var email, name string
	err := s.db.QueryRow(r.Context(), `
		SELECT id, email, name FROM users WHERE lower(email) = lower($1) AND active`,
		strings.TrimSpace(in.Email)).Scan(&userID, &email, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		internalError(w, err)
		return
	}
	token := hex.EncodeToString(raw[:])
	link := s.appURL + "/reset-password?token=" + url.QueryEscape(token)

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if _, err := tx.Exec(r.Context(), `
		INSERT INTO password_resets (token_hash, user_id, expires_at) VALUES ($1, $2, now() + make_interval(mins => $3))`,
		hashResetToken(token), userID, int(resetTokenTTL.Minutes())); err != nil {
		internalError(w, err)
		return
	}
	if _, err := tx.Exec(r.Context(), `
		INSERT INTO email_outbox (to_email, subject, body) VALUES ($1, $2, $3)`,
		email, "Restablece tu contraseña de MiColmena",
		"Hola "+name+":\n\nPara elegir una contraseña nueva, abre este enlace (vale durante 1 hora):\n\n"+link+
			"\n\nSi no lo pediste tú, ignora este correo: tu contraseña no cambia."); err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// resetPassword cambia la contraseña con el token del correo e inicia sesión.
func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	errs := validationErrors{}
	checkPassword(errs, "password", in.Password)
	if errs.write(w) {
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		internalError(w, err)
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())

	var userID int64
	err = tx.QueryRow(r.Context(), `
		UPDATE password_resets SET used_at = now()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()
		RETURNING user_id`, hashResetToken(strings.TrimSpace(in.Token))).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		validationErrors{"token": "el enlace no es válido o ya caducó; pide otro"}.write(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	u, err := scanUser(tx.QueryRow(r.Context(), `
		UPDATE users SET password_hash = $2, tokens_valid_after = date_trunc('second', now())
		WHERE id = $1 AND active
		RETURNING `+userColumns, userID, hash))
	if errors.Is(err, pgx.ErrNoRows) {
		validationErrors{"token": "la cuenta está desactivada"}.write(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	// Los demás enlaces pendientes de esta cuenta dejan de valer.
	if _, err := tx.Exec(r.Context(), `
		UPDATE password_resets SET used_at = now() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	s.loginLock.Reset(strings.ToLower(u.Email))
	s.respondWithToken(w, http.StatusOK, u)
}
