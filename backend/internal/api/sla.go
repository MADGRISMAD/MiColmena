package api

import (
	"net/http"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

type SLAPolicy struct {
	Priority             string `json:"priority"`
	FirstResponseMinutes int    `json:"first_response_minutes"`
	ResolutionMinutes    int    `json:"resolution_minutes"`
}

func (s *Server) listSLA(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `
		SELECT priority, first_response_minutes, resolution_minutes FROM sla_policies
		WHERE org_id = $1
		ORDER BY array_position(ARRAY['urgent', 'high', 'medium', 'low'], priority)`, claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SLAPolicy])
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

const maxSLAMinutes = 60 * 24 * 90 // 90 días

// updateSLA reemplaza los plazos de las prioridades enviadas.
func (s *Server) updateSLA(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Items []SLAPolicy `json:"items"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	errs := validationErrors{}
	errs.check(len(in.Items) > 0, "items", "envía al menos una prioridad")
	seen := map[string]bool{}
	for _, p := range in.Items {
		field := "items." + p.Priority
		errs.check(slices.Contains(ticketPriorities, p.Priority) && !seen[p.Priority], "items", "prioridad inválida o repetida")
		seen[p.Priority] = true
		errs.check(p.FirstResponseMinutes > 0 && p.FirstResponseMinutes <= maxSLAMinutes, field, "primera respuesta fuera de rango")
		errs.check(p.ResolutionMinutes > 0 && p.ResolutionMinutes <= maxSLAMinutes, field, "resolución fuera de rango")
		errs.check(p.ResolutionMinutes >= p.FirstResponseMinutes, field, "la resolución no puede ser antes que la primera respuesta")
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
	for _, p := range in.Items {
		if _, err := tx.Exec(r.Context(), `
			UPDATE sla_policies SET first_response_minutes = $2, resolution_minutes = $3
			WHERE priority = $1 AND org_id = $4`,
			p.Priority, p.FirstResponseMinutes, p.ResolutionMinutes, claimsFrom(r).OrgID); err != nil {
			internalError(w, err)
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	s.listSLA(w, r)
}

// rateTicket guarda la valoración del cliente. Solo el solicitante, y solo con el ticket resuelto o cerrado.
func (s *Server) rateTicket(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	ticket, ok := s.loadTicket(w, r, id)
	if !ok {
		return
	}
	claims := claimsFrom(r)
	if ticket.Requester.ID != claims.UserID() {
		writeError(w, http.StatusForbidden, "solo quien abrió el ticket puede valorarlo")
		return
	}
	var in struct {
		Rating  string `json:"rating"`
		Comment string `json:"comment"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Comment = strings.TrimSpace(in.Comment)
	errs := validationErrors{}
	errs.check(in.Rating == "good" || in.Rating == "bad", "rating", "elige good o bad")
	errs.check(len(in.Comment) <= 2000, "comment", "es demasiado largo")
	errs.check(ticket.Status == "resolved" || ticket.Status == "closed", "rating", "el ticket aún no está resuelto")
	if errs.write(w) {
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if _, err := tx.Exec(r.Context(), `
		UPDATE tickets SET satisfaction = $2, satisfaction_comment = $3, rated_at = now() WHERE id = $1`,
		id, in.Rating, in.Comment); err != nil {
		internalError(w, err)
		return
	}
	if ticket.Satisfaction != in.Rating {
		if err := addEvent(r.Context(), tx, id, claims.UserID(), "satisfaction", ticket.Satisfaction, in.Rating); err != nil {
			internalError(w, err)
			return
		}
	}
	t, err := fetchTicket(r.Context(), tx, id)
	if err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	var out fanout
	out.ticket(t, true)
	s.broker.publish(out)
	writeJSON(w, http.StatusOK, t)
}
