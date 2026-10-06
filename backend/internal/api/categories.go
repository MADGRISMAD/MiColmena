package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Category struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT id, name, created_at FROM categories ORDER BY lower(name)`)
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Category])
	if err != nil {
		internalError(w, err)
		return
	}
	if items == nil {
		items = []Category{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func readCategoryName(w http.ResponseWriter, r *http.Request) (string, bool) {
	var in struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &in) {
		return "", false
	}
	name := strings.TrimSpace(in.Name)
	errs := validationErrors{}
	errs.check(name != "", "name", "es obligatorio")
	errs.check(len(name) <= 100, "name", "debe tener como máximo 100 caracteres")
	if errs.write(w) {
		return "", false
	}
	return name, true
}

func (s *Server) createCategory(w http.ResponseWriter, r *http.Request) {
	name, ok := readCategoryName(w, r)
	if !ok {
		return
	}
	var c Category
	err := s.db.QueryRow(r.Context(),
		`INSERT INTO categories (name) VALUES ($1) RETURNING id, name, created_at`, name,
	).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if isUniqueViolation(err) {
		validationErrors{"name": "ya existe una categoría con ese nombre"}.write(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// renameCategory cambia el nombre también en los tickets que la usan.
func (s *Server) renameCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	name, ok := readCategoryName(w, r)
	if !ok {
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())

	var old string
	if err := tx.QueryRow(r.Context(), `SELECT name FROM categories WHERE id = $1 FOR UPDATE`, id).Scan(&old); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "categoría no encontrada")
			return
		}
		internalError(w, err)
		return
	}
	var c Category
	err = tx.QueryRow(r.Context(),
		`UPDATE categories SET name = $2 WHERE id = $1 RETURNING id, name, created_at`, id, name,
	).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if isUniqueViolation(err) {
		validationErrors{"name": "ya existe una categoría con ese nombre"}.write(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if _, err := tx.Exec(r.Context(), `UPDATE tickets SET category = $2 WHERE category = $1`, old, name); err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// deleteCategory la quita de la lista; los tickets que la usaban la conservan.
func (s *Server) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	tag, err := s.db.Exec(r.Context(), `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		internalError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "categoría no encontrada")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
