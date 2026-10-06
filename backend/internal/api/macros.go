package api

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Macro es una respuesta guardada. Las variables {{solicitante}}, {{agente}} y {{ticket}}
// las reemplaza el frontend al insertarla.
type Macro struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	Internal  bool      `json:"internal"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const macroColumns = `id, title, body, status, internal, created_at, updated_at`

func (s *Server) listMacros(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT `+macroColumns+` FROM macros ORDER BY lower(title)`)
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Macro])
	if err != nil {
		internalError(w, err)
		return
	}
	if items == nil {
		items = []Macro{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type macroInput struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Status   string `json:"status"`
	Internal bool   `json:"internal"`
}

func readMacro(w http.ResponseWriter, r *http.Request) (macroInput, bool) {
	var in macroInput
	if !decodeJSON(w, r, &in) {
		return in, false
	}
	in.Title = strings.TrimSpace(in.Title)
	errs := validationErrors{}
	errs.check(in.Title != "", "title", "es obligatorio")
	errs.check(len(in.Title) <= 100, "title", "debe tener como máximo 100 caracteres")
	errs.check(notBlank(in.Body), "body", "es obligatorio")
	errs.check(len(in.Body) <= 20000, "body", "es demasiado largo")
	errs.check(in.Status == "" || slices.Contains(ticketStatuses, in.Status), "status", "valor inválido")
	return in, !errs.write(w)
}

func (s *Server) createMacro(w http.ResponseWriter, r *http.Request) {
	in, ok := readMacro(w, r)
	if !ok {
		return
	}
	m, err := scanMacro(s.db.QueryRow(r.Context(), `
		INSERT INTO macros (title, body, status, internal, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+macroColumns, in.Title, in.Body, in.Status, in.Internal, claimsFrom(r).UserID()))
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) updateMacro(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	in, ok := readMacro(w, r)
	if !ok {
		return
	}
	m, err := scanMacro(s.db.QueryRow(r.Context(), `
		UPDATE macros SET title = $2, body = $3, status = $4, internal = $5, updated_at = now()
		WHERE id = $1
		RETURNING `+macroColumns, id, in.Title, in.Body, in.Status, in.Internal))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "respuesta guardada no encontrada")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) deleteMacro(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	tag, err := s.db.Exec(r.Context(), `DELETE FROM macros WHERE id = $1`, id)
	if err != nil {
		internalError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "respuesta guardada no encontrada")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func scanMacro(row pgx.Row) (Macro, error) {
	var m Macro
	err := row.Scan(&m.ID, &m.Title, &m.Body, &m.Status, &m.Internal, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}
