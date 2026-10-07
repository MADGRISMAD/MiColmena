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
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
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
	Tags         []string        `json:"tags"`
	CustomFields json.RawMessage `json:"custom_fields"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ResolvedAt   *time.Time      `json:"resolved_at"`

	// SLA: plazos calculados desde la creación según la prioridad actual.
	FirstResponseAt  *time.Time `json:"first_response_at"`
	FirstResponseDue time.Time  `json:"first_response_due"`
	ResolutionDue    time.Time  `json:"resolution_due"`

	// Encuesta de satisfacción: "", "good" o "bad".
	Satisfaction        string     `json:"satisfaction"`
	SatisfactionComment string     `json:"satisfaction_comment"`
	RatedAt             *time.Time `json:"rated_at"`

	// OrgID es la empresa del ticket; no se envía al cliente.
	OrgID int64 `json:"-"`
}

// ticketSelect une al solicitante y al agente asignado en una sola consulta (sin N+1).
const ticketSelect = `
	SELECT t.id, t.title, t.description, t.status, t.priority, t.category,
	       r.id, r.name, a.id, a.name, t.tags,
	       t.custom_fields, t.created_at, t.updated_at, t.resolved_at,
	       t.first_response_at,
	       t.created_at + p.first_response_minutes * interval '1 minute',
	       t.created_at + p.resolution_minutes * interval '1 minute',
	       t.satisfaction, t.satisfaction_comment, t.rated_at, t.org_id
	FROM tickets t
	JOIN users r ON r.id = t.requester_id
	LEFT JOIN users a ON a.id = t.assignee_id
	JOIN sla_policies p ON p.org_id = t.org_id AND p.priority = t.priority`

// slaBreached es la condición SQL de un ticket pendiente que ya superó alguno de sus plazos.
const slaBreached = `(t.status NOT IN ('resolved', 'closed') AND (
		(t.first_response_at IS NULL AND t.created_at + p.first_response_minutes * interval '1 minute' < now())
		OR t.created_at + p.resolution_minutes * interval '1 minute' < now()))`

func scanTicket(row pgx.Row) (Ticket, error) {
	var t Ticket
	var assigneeID *int64
	var assigneeName *string
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.Category,
		&t.Requester.ID, &t.Requester.Name, &assigneeID, &assigneeName, &t.Tags,
		&t.CustomFields, &t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt,
		&t.FirstResponseAt, &t.FirstResponseDue, &t.ResolutionDue,
		&t.Satisfaction, &t.SatisfactionComment, &t.RatedAt, &t.OrgID)
	if assigneeID != nil {
		t.Assignee = &UserRef{ID: *assigneeID, Name: *assigneeName}
	}
	return t, err
}

// listTickets admite filtros y paginación por cursor:
//
//	?status=open&priority=high&assignee=me|none|<id>&tag=vip&sla=breached&q=texto&before=<id>&limit=25
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

	// Cada empresa solo ve lo suyo.
	where = append(where, "t.org_id = "+arg(claims.OrgID))
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
	switch q.Get("sla") {
	case "":
	case "breached":
		where = append(where, slaBreached)
	default:
		writeError(w, http.StatusBadRequest, "sla inválido")
		return
	}
	if v := q.Get("tag"); v != "" {
		where = append(where, "t.tags @> ARRAY["+arg(normalizeTag(v))+"]::text[]")
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
	// Un ticket de otra empresa se trata igual que uno inexistente, para no revelar que existe.
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (t.OrgID != claims.OrgID ||
		(!claims.IsStaff() && t.Requester.ID != claims.UserID()))) {
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

	category, err := s.canonicalCategory(r.Context(), claims.OrgID, in.Category)
	if err != nil {
		internalError(w, err)
		return
	}
	if category == nil {
		validationErrors{"category": "no es una categoría válida"}.write(w)
		return
	}

	requester := claims.UserID()
	if in.RequesterID != nil {
		requester = *in.RequesterID
		if exists, err := s.userExists(r.Context(), claims.OrgID, requester, false); err != nil {
			internalError(w, err)
			return
		} else if !exists {
			validationErrors{"requester_id": "el usuario no existe"}.write(w)
			return
		}
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())

	var id int64
	err = tx.QueryRow(r.Context(), `
		INSERT INTO tickets (title, description, priority, category, custom_fields, requester_id, org_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		in.Title, in.Description, in.Priority, *category, in.CustomFields, requester, claims.OrgID,
	).Scan(&id)
	if err != nil {
		internalError(w, err)
		return
	}
	if err := addEvent(r.Context(), tx, id, claims.UserID(), "created", "", ""); err != nil {
		internalError(w, err)
		return
	}
	t, err := fetchTicket(r.Context(), tx, id)
	if err != nil {
		internalError(w, err)
		return
	}
	var out fanout
	out.ticket(t, false)
	if err := s.notifyNewTicket(r.Context(), tx, &out, claims.UserID(), t); err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	s.broker.publish(out)
	writeJSON(w, http.StatusCreated, t)
}

// ticketPatch son los cambios de un PATCH. Solo se aplican los campos enviados.
type ticketPatch struct {
	Title        *string         `json:"title"`
	Description  *string         `json:"description"`
	Status       *string         `json:"status"`
	Priority     *string         `json:"priority"`
	Category     *string         `json:"category"`
	CustomFields json.RawMessage `json:"custom_fields"`
	// null desasigna el ticket.
	AssigneeID json.RawMessage `json:"assignee_id"`
	Tags       *[]string       `json:"tags"`

	// Rellenados por validate.
	assigneeSet bool
	assignee    *int64
}

// validate normaliza el patch y comprueba permisos y valores. Consulta la base para
// validar el agente asignado y la categoría.
func (s *Server) validatePatch(ctx context.Context, claims auth.Claims, p *ticketPatch) (validationErrors, error) {
	errs := validationErrors{}
	staffOnly := func(field string, present bool) {
		errs.check(!present || claims.IsStaff(), field, "solo un agente puede cambiarlo")
	}
	staffOnly("priority", p.Priority != nil)
	staffOnly("category", p.Category != nil)
	staffOnly("custom_fields", p.CustomFields != nil)
	staffOnly("assignee_id", p.AssigneeID != nil)
	staffOnly("tags", p.Tags != nil)
	// Un cliente solo puede cerrar su propio ticket, no moverlo a otros estados.
	errs.check(p.Status == nil || claims.IsStaff() || *p.Status == "closed", "status", "solo puedes cerrar el ticket")

	if p.Title != nil {
		title := strings.TrimSpace(*p.Title)
		errs.check(notBlank(title), "title", "es obligatorio")
		errs.check(len(title) <= 200, "title", "debe tener como máximo 200 caracteres")
		p.Title = &title
	}
	if p.Description != nil {
		errs.check(len(*p.Description) <= 20000, "description", "es demasiado larga")
	}
	if p.Status != nil {
		errs.check(slices.Contains(ticketStatuses, *p.Status), "status", "valor inválido")
	}
	if p.Priority != nil {
		errs.check(slices.Contains(ticketPriorities, *p.Priority), "priority", "valor inválido")
	}
	if p.CustomFields != nil {
		errs.check(isJSONObject(p.CustomFields), "custom_fields", "debe ser un objeto JSON")
	}
	if p.Tags != nil {
		tags, msg := normalizeTags(*p.Tags)
		errs.check(msg == "", "tags", msg)
		p.Tags = &tags
	}
	if len(errs) > 0 {
		return errs, nil
	}

	if p.Category != nil {
		category, err := s.canonicalCategory(ctx, claims.OrgID, *p.Category)
		if err != nil {
			return nil, err
		}
		errs.check(category != nil, "category", "no es una categoría válida")
		p.Category = category
	}
	if p.AssigneeID != nil {
		p.assigneeSet = true
		if string(p.AssigneeID) != "null" {
			var assignee int64
			if err := json.Unmarshal(p.AssigneeID, &assignee); err != nil {
				errs.check(false, "assignee_id", "debe ser un número o null")
			} else if exists, err := s.userExists(ctx, claims.OrgID, assignee, true); err != nil {
				return nil, err
			} else {
				errs.check(exists, "assignee_id", "debe ser un agente o administrador activo")
				p.assignee = &assignee
			}
		}
	}
	return errs, nil
}

// applyPatch guarda los cambios en la transacción y registra el historial.
// current debe estar bloqueado (fetchTicketForUpdate) para que el historial sea coherente.
func applyPatch(ctx context.Context, tx pgx.Tx, actorID int64, current Ticket, p ticketPatch) (Ticket, error) {
	var sets []string
	var args []any
	set := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if p.Title != nil {
		set("title", *p.Title)
	}
	if p.Description != nil {
		set("description", *p.Description)
	}
	if p.Status != nil {
		set("status", *p.Status)
		resolved := *p.Status == "resolved" || *p.Status == "closed"
		wasResolved := current.Status == "resolved" || current.Status == "closed"
		if resolved && !wasResolved {
			sets = append(sets, "resolved_at = now()")
		} else if !resolved {
			sets = append(sets, "resolved_at = NULL")
		}
	}
	if p.Priority != nil {
		set("priority", *p.Priority)
	}
	if p.Category != nil {
		set("category", *p.Category)
	}
	if p.CustomFields != nil {
		set("custom_fields", p.CustomFields)
	}
	if p.assigneeSet {
		set("assignee_id", p.assignee)
	}
	if p.Tags != nil {
		set("tags", *p.Tags)
	}
	if len(sets) == 0 {
		return current, nil
	}

	sets = append(sets, "updated_at = now()")
	args = append(args, current.ID)
	sql := fmt.Sprintf("UPDATE tickets SET %s WHERE id = $%d", strings.Join(sets, ", "), len(args))
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return Ticket{}, err
	}
	updated, err := fetchTicket(ctx, tx, current.ID)
	if err != nil {
		return Ticket{}, err
	}
	return updated, recordChanges(ctx, tx, actorID, current, updated)
}

// updateTicket aplica solo los campos enviados. Para desasignar, enviar "assignee_id": null.
func (s *Server) updateTicket(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := s.loadTicket(w, r, id); !ok {
		return
	}
	claims := claimsFrom(r)

	var in ticketPatch
	if !decodeJSON(w, r, &in) {
		return
	}
	errs, err := s.validatePatch(r.Context(), claims, &in)
	if err != nil {
		internalError(w, err)
		return
	}
	if errs.write(w) {
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())

	current, err := fetchTicketForUpdate(r.Context(), tx, claims.OrgID, id)
	if err != nil {
		internalError(w, err)
		return
	}
	updated, err := applyPatch(r.Context(), tx, claims.UserID(), current, in)
	if err != nil {
		internalError(w, err)
		return
	}
	var out fanout
	out.ticket(updated, false)
	if err := s.notifyChanges(r.Context(), tx, &out, claims.UserID(), current, updated); err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	s.broker.publish(out)
	writeJSON(w, http.StatusOK, updated)
}

const maxBulkTickets = 100

// bulkUpdateTickets aplica el mismo cambio a varios tickets a la vez (solo agentes).
func (s *Server) bulkUpdateTickets(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var in struct {
		IDs        []int64         `json:"ids"`
		Status     *string         `json:"status"`
		Priority   *string         `json:"priority"`
		AssigneeID json.RawMessage `json:"assignee_id"`
		AddTags    []string        `json:"add_tags"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}

	errs := validationErrors{}
	errs.check(len(in.IDs) > 0, "ids", "elige al menos un ticket")
	errs.check(len(in.IDs) <= maxBulkTickets, "ids", fmt.Sprintf("como máximo %d tickets a la vez", maxBulkTickets))
	addTags, msg := normalizeTags(in.AddTags)
	errs.check(msg == "", "add_tags", msg)
	if errs.write(w) {
		return
	}

	p := ticketPatch{Status: in.Status, Priority: in.Priority, AssigneeID: in.AssigneeID}
	errs, err := s.validatePatch(r.Context(), claims, &p)
	if err != nil {
		internalError(w, err)
		return
	}
	if errs.write(w) {
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())

	ids := slices.Clone(in.IDs)
	slices.Sort(ids) // Mismo orden de bloqueo en todas las peticiones: evita interbloqueos.
	ids = slices.Compact(ids)
	updated := 0
	var out fanout
	for _, id := range ids {
		// Los ids de otra empresa se ignoran como si no existieran.
		current, err := fetchTicketForUpdate(r.Context(), tx, claims.OrgID, id)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			internalError(w, err)
			return
		}
		patch := p
		if len(addTags) > 0 {
			merged, msg := normalizeTags(append(slices.Clone(current.Tags), addTags...))
			if msg != "" {
				validationErrors{"add_tags": fmt.Sprintf("el ticket #%d: %s", id, msg)}.write(w)
				return
			}
			patch.Tags = &merged
		}
		after, err := applyPatch(r.Context(), tx, claims.UserID(), current, patch)
		if err != nil {
			internalError(w, err)
			return
		}
		out.ticket(after, false)
		if err := s.notifyChanges(r.Context(), tx, &out, claims.UserID(), current, after); err != nil {
			internalError(w, err)
			return
		}
		updated++
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	s.broker.publish(out)
	writeJSON(w, http.StatusOK, map[string]int{"updated": updated})
}

// listTags devuelve las etiquetas en uso con cuántos tickets las tienen.
func (s *Server) listTags(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `
		SELECT tag, count(*) FROM tickets, unnest(tags) AS tag
		WHERE org_id = $1
		GROUP BY tag ORDER BY count(*) DESC, tag
		LIMIT 200`, claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	type tagCount struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	tags, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (tagCount, error) {
		var t tagCount
		err := row.Scan(&t.Name, &t.Count)
		return t, err
	})
	if err != nil {
		internalError(w, err)
		return
	}
	if tags == nil {
		tags = []tagCount{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tags})
}

const (
	maxTags      = 10
	maxTagLength = 30
)

// normalizeTag pasa a minúsculas y cambia los espacios por guiones: "Cliente VIP" → "cliente-vip".
func normalizeTag(tag string) string {
	return strings.Join(strings.Fields(strings.ToLower(tag)), "-")
}

// normalizeTags normaliza y quita duplicados. Devuelve un mensaje si algo no es válido.
func normalizeTags(in []string) ([]string, string) {
	out := []string{}
	for _, raw := range in {
		tag := normalizeTag(raw)
		if tag == "" || slices.Contains(out, tag) {
			continue
		}
		if len([]rune(tag)) > maxTagLength {
			return nil, fmt.Sprintf("cada etiqueta debe tener como máximo %d caracteres", maxTagLength)
		}
		out = append(out, tag)
	}
	if len(out) > maxTags {
		return nil, fmt.Sprintf("como máximo %d etiquetas", maxTags)
	}
	return out, ""
}

// querier es lo que comparten el pool y una transacción.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func fetchTicket(ctx context.Context, q querier, id int64) (Ticket, error) {
	return scanTicket(q.QueryRow(ctx, ticketSelect+" WHERE t.id = $1", id))
}

// fetchTicketForUpdate lee el ticket y bloquea su fila hasta el final de la transacción.
func fetchTicketForUpdate(ctx context.Context, tx pgx.Tx, orgID, id int64) (Ticket, error) {
	return scanTicket(tx.QueryRow(ctx, ticketSelect+" WHERE t.id = $1 AND t.org_id = $2 FOR UPDATE OF t", id, orgID))
}

// canonicalCategory devuelve el nombre tal como está en la lista de categorías, "" si viene vacía,
// o nil si no existe.
func (s *Server) canonicalCategory(ctx context.Context, orgID int64, category string) (*string, error) {
	category = strings.TrimSpace(category)
	if category == "" {
		return &category, nil
	}
	var name string
	err := s.db.QueryRow(ctx, `SELECT name FROM categories WHERE org_id = $1 AND lower(name) = lower($2)`, orgID, category).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &name, nil
}

// userExists comprueba si existe el usuario activo en la empresa; con staff=true exige que sea agente o administrador.
func (s *Server) userExists(ctx context.Context, orgID, id int64, staff bool) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users
			WHERE id = $1 AND org_id = $3 AND active AND (NOT $2 OR role IN ('agent', 'admin'))
		)`, id, staff, orgID).Scan(&exists)
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
