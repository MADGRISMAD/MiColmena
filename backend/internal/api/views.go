package api

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// SavedView es una combinación de filtros de la lista de tickets, guardada por un usuario.
type SavedView struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`
	CreatedAt time.Time `json:"created_at"`
}

const maxSavedViews = 20

func (s *Server) listViews(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(),
		`SELECT id, name, query, created_at FROM saved_views WHERE user_id = $1 ORDER BY id`, claimsFrom(r).UserID())
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SavedView])
	if err != nil {
		internalError(w, err)
		return
	}
	if items == nil {
		items = []SavedView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// viewParams son los filtros que puede guardar una vista.
var viewParams = map[string]bool{"status": true, "priority": true, "assignee": true, "tag": true, "sla": true, "q": true}

func (s *Server) createView(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name  string `json:"name"`
		Query string `json:"query"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	values, err := url.ParseQuery(strings.TrimPrefix(in.Query, "?"))
	errs := validationErrors{}
	errs.check(in.Name != "", "name", "es obligatorio")
	errs.check(len(in.Name) <= 60, "name", "debe tener como máximo 60 caracteres")
	errs.check(err == nil && len(in.Query) <= 500, "query", "filtros inválidos")
	for key := range values {
		errs.check(viewParams[key], "query", "filtro desconocido: "+key)
	}
	errs.check(len(values) > 0, "query", "elige al menos un filtro")
	if errs.write(w) {
		return
	}

	userID := claimsFrom(r).UserID()
	var count int
	if err := s.db.QueryRow(r.Context(), `SELECT count(*) FROM saved_views WHERE user_id = $1`, userID).Scan(&count); err != nil {
		internalError(w, err)
		return
	}
	if count >= maxSavedViews {
		validationErrors{"name": "ya tienes el máximo de vistas guardadas"}.write(w)
		return
	}

	var v SavedView
	if err := s.db.QueryRow(r.Context(), `
		INSERT INTO saved_views (user_id, name, query) VALUES ($1, $2, $3)
		RETURNING id, name, query, created_at`, userID, in.Name, "?"+values.Encode(),
	).Scan(&v.ID, &v.Name, &v.Query, &v.CreatedAt); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) deleteView(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	tag, err := s.db.Exec(r.Context(), `DELETE FROM saved_views WHERE id = $1 AND user_id = $2`, id, claimsFrom(r).UserID())
	if err != nil {
		internalError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "vista no encontrada")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
