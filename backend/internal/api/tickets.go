package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ticketStatuses   = []string{"open", "in_progress", "waiting", "resolved", "closed"}
	ticketPriorities = []string{"low", "medium", "high", "urgent"}
)

const (
	defaultPageSize = 25
	maxPageSize     = 100
)

type UserRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Ticket struct {
	ID           int64           `json:"id"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Status       string          `json:"status"`
	Priority     string          `json:"priority"`
	Category     string          `json:"category"`
	Requester    UserRef         `json:"requester"`
	Assignee     *UserRef        `json:"assignee"`
	CustomFields json.RawMessage `json:"custom_fields"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ResolvedAt   *time.Time      `json:"resolved_at"`
}

// ticketSelect une al solicitante y al agente asignado en una sola consulta (sin N+1).
const ticketSelect = `
	SELECT t.id, t.title, t.description, t.status, t.priority, t.category,
	       r.id, r.name, a.id, a.name,
	       t.custom_fields, t.created_at, t.updated_at, t.resolved_at
	FROM tickets t
	JOIN users r ON r.id = t.requester_id
	LEFT JOIN users a ON a.id = t.assignee_id`

func scanTicket(row pgx.Row) (Ticket, error) {
	var t Ticket
	var assigneeID *int64
	var assigneeName *string
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.Category,
		&t.Requester.ID, &t.Requester.Name, &assigneeID, &assigneeName,
		&t.CustomFields, &t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt)
	if assigneeID != nil {
		t.Assignee = &UserRef{ID: *assigneeID, Name: *assigneeName}
	}
	return t, err
}

// listTickets admite filtros y paginación por cursor:
//
//	?status=open&priority=high&assignee=me|none|<id>&q=texto&before=<id>&limit=25
//
// La paginación por cursor (id < before) se mantiene rápida aunque haya millones de
// tickets, a diferencia de OFFSET, que recorre todas las filas saltadas.
func (s *Server) listTickets(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	q := r.URL.Query()

	var where []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	if !claims.IsStaff() {
		where = append(where, "t.requester_id = "+arg(claims.UserID()))
	}
	if v := q.Get("status"); v != "" {
		if !slices.Contains(ticketStatuses, v) {
			writeError(w, http.StatusBadRequest, "status inválido")
			return
		}
		where = append(where, "t.status = "+arg(v))
	}
	if v := q.Get("priority"); v != "" {
		if !slices.Contains(ticketPriorities, v) {
			writeError(w, http.StatusBadRequest, "priority inválida")
			return
		}
		where = append(where, "t.priority = "+arg(v))
	}
	switch v := q.Get("assignee"); v {
	case "":
	case "me":
		where = append(where, "t.assignee_id = "+arg(claims.UserID()))
	case "none":
		where = append(where, "t.assignee_id IS NULL")
	default:
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "assignee inválido")
			return
		}
		where = append(where, "t.assignee_id = "+arg(id))
	}
	if v := strings.TrimSpace(q.Get("q")); v != "" {
		where = append(where, "t.search @@ websearch_to_tsquery('spanish', "+arg(v)+")")
	}
	if v := q.Get("before"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "before inválido")
			return
		}
		where = append(where, "t.id < "+arg(id))
	}

	limit := defaultPageSize
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxPageSize {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("limit debe estar entre 1 y %d", maxPageSize))
			return
		}
		limit = n
	}

	sql := ticketSelect
	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}
	// Se pide una fila extra para saber si hay otra página.
	sql += " ORDER BY t.id DESC LIMIT " + arg(limit+1)

	rows, err := s.db.Query(r.Context(), sql, args...)
	if err != nil {
		internalError(w, err)
		return
	}
	tickets, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Ticket, error) { return scanTicket(row) })
	if err != nil {
		internalError(w, err)
		return
	}

	var next *int64
	if len(tickets) > limit {
		tickets = tickets[:limit]
		next = &tickets[limit-1].ID
	}
	if tickets == nil {
		tickets = []Ticket{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tickets, "next_cursor": next})
}

// loadTicket devuelve el ticket si el usuario puede verlo. Los clientes solo ven los suyos;
// para los ajenos se responde 404 para no revelar que existen.
func (s *Server) loadTicket(w http.ResponseWriter, r *http.Request, id int64) (Ticket, bool) {
	t, err := scanTicket(s.db.QueryRow(r.Context(), ticketSelect+" WHERE t.id = $1", id))
	claims := claimsFrom(r)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !claims.IsStaff() && t.Requester.ID != claims.UserID()) {
		writeError(w, http.StatusNotFound, "ticket no encontrado")
		return Ticket{}, false
	}
	if err != nil {
		internalError(w, err)
		return Ticket{}, false
	}
	return t, true
}

func (s *Server) getTicket(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if t, ok := s.loadTicket(w, r, id); ok {
		writeJSON(w, http.StatusOK, t)
	}
}

func (s *Server) createTicket(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var in struct {
		Title        string          `json:"title"`
		Description  string          `json:"description"`
		Priority     string          `json:"priority"`
		Category     string          `json:"category"`
		CustomFields json.RawMessage `json:"custom_fields"`
		// Solo agentes y administradores pueden abrir tickets en nombre de otro usuario.
		RequesterID *int64 `json:"requester_id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Priority == "" {
		in.Priority = "medium"
	}
	if len(in.CustomFields) == 0 || string(in.CustomFields) == "null" {
		in.CustomFields = json.RawMessage("{}")
	}

	errs := validationErrors{}
	errs.check(notBlank(in.Title), "title", "es obligatorio")
	errs.check(len(in.Title) <= 200, "title", "debe tener como máximo 200 caracteres")
	errs.check(len(in.Description) <= 20000, "description", "es demasiado larga")
	errs.check(slices.Contains(ticketPriorities, in.Priority), "priority", "valor inválido")
	errs.check(len(in.Category) <= 100, "category", "debe tener como máximo 100 caracteres")
	errs.check(isJSONObject(in.CustomFields), "custom_fields", "debe ser un objeto JSON")
	errs.check(in.RequesterID == nil || claims.IsStaff(), "requester_id", "no permitido")
	if errs.write(w) {
		return
	}

	requester := claims.UserID()
	if in.RequesterID != nil {
		requester = *in.RequesterID
		if exists, err := s.userExists(r.Context(), requester, false); err != nil {
			internalError(w, err)
			return
		} else if !exists {
			validationErrors{"requester_id": "el usuario no existe"}.write(w)
			return
		}
	}

	var id int64
	err := s.db.QueryRow(r.Context(), `
		INSERT INTO tickets (title, description, priority, category, custom_fields, requester_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		in.Title, in.Description, in.Priority, strings.TrimSpace(in.Category), in.CustomFields, requester,
	).Scan(&id)
	if err != nil {
		internalError(w, err)
		return
	}

	t, err := scanTicket(s.db.QueryRow(r.Context(), ticketSelect+" WHERE t.id = $1", id))
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// updateTicket aplica solo los campos enviados. Para desasignar, enviar "assignee_id": null.
func (s *Server) updateTicket(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	current, ok := s.loadTicket(w, r, id)
	if !ok {
		return
	}
	claims := claimsFrom(r)

	var in struct {
		Title        *string         `json:"title"`
		Description  *string         `json:"description"`
		Status       *string         `json:"status"`
		Priority     *string         `json:"priority"`
		Category     *string         `json:"category"`
		CustomFields json.RawMessage `json:"custom_fields"`
		AssigneeID   json.RawMessage `json:"assignee_id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}

	var sets []string
	var args []any
	set := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	errs := validationErrors{}
	staffOnly := func(field string, present bool) {
		errs.check(!present || claims.IsStaff(), field, "solo un agente puede cambiarlo")
	}
	staffOnly("priority", in.Priority != nil)
	staffOnly("category", in.Category != nil)
	staffOnly("custom_fields", in.CustomFields != nil)
	staffOnly("assignee_id", in.AssigneeID != nil)
	// Un cliente solo puede cerrar su propio ticket, no moverlo a otros estados.
	errs.check(in.Status == nil || claims.IsStaff() || *in.Status == "closed", "status", "solo puedes cerrar el ticket")

	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		errs.check(notBlank(title), "title", "es obligatorio")
		errs.check(len(title) <= 200, "title", "debe tener como máximo 200 caracteres")
		set("title", title)
	}
	if in.Description != nil {
		errs.check(len(*in.Description) <= 20000, "description", "es demasiado larga")
		set("description", *in.Description)
	}
	if in.Status != nil {
		errs.check(slices.Contains(ticketStatuses, *in.Status), "status", "valor inválido")
		set("status", *in.Status)
		resolved := *in.Status == "resolved" || *in.Status == "closed"
		wasResolved := current.Status == "resolved" || current.Status == "closed"
		if resolved && !wasResolved {
			sets = append(sets, "resolved_at = now()")
		} else if !resolved {
			sets = append(sets, "resolved_at = NULL")
		}
	}
	if in.Priority != nil {
		errs.check(slices.Contains(ticketPriorities, *in.Priority), "priority", "valor inválido")
		set("priority", *in.Priority)
	}
	if in.Category != nil {
		errs.check(len(*in.Category) <= 100, "category", "debe tener como máximo 100 caracteres")
		set("category", strings.TrimSpace(*in.Category))
	}
	if in.CustomFields != nil {
		errs.check(isJSONObject(in.CustomFields), "custom_fields", "debe ser un objeto JSON")
		set("custom_fields", in.CustomFields)
	}
	if in.AssigneeID != nil && claims.IsStaff() {
		if string(in.AssigneeID) == "null" {
			set("assignee_id", nil)
		} else {
			var assignee int64
			if err := json.Unmarshal(in.AssigneeID, &assignee); err != nil {
				errs.check(false, "assignee_id", "debe ser un número o null")
			} else if exists, err := s.userExists(r.Context(), assignee, true); err != nil {
				internalError(w, err)
				return
			} else {
				errs.check(exists, "assignee_id", "debe ser un agente o administrador")
				set("assignee_id", assignee)
			}
		}
	}
	if errs.write(w) {
		return
	}
	if len(sets) == 0 {
		writeJSON(w, http.StatusOK, current)
		return
	}

	sets = append(sets, "updated_at = now()")
	args = append(args, id)
	sql := fmt.Sprintf("UPDATE tickets SET %s WHERE id = $%d", strings.Join(sets, ", "), len(args))
	if _, err := s.db.Exec(r.Context(), sql, args...); err != nil {
		internalError(w, err)
		return
	}

	t, err := scanTicket(s.db.QueryRow(r.Context(), ticketSelect+" WHERE t.id = $1", id))
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// userExists comprueba si existe el usuario; con staff=true exige que sea agente o administrador.
func (s *Server) userExists(ctx context.Context, id int64, staff bool) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users WHERE id = $1 AND (NOT $2 OR role IN ('agent', 'admin'))
		)`, id, staff).Scan(&exists)
	return exists, err
}

func isJSONObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return false
	}
	var obj map[string]any
	return json.Unmarshal(trimmed, &obj) == nil
}
