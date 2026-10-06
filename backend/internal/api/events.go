package api

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// TicketEvent es una entrada del historial de un ticket.
type TicketEvent struct {
	ID        int64     `json:"id"`
	Actor     *UserRef  `json:"actor"`
	Kind      string    `json:"kind"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	CreatedAt time.Time `json:"created_at"`
}

// publicEventKinds son los cambios que también ve el cliente. La prioridad, la asignación,
// la categoría y las etiquetas son organización interna del equipo.
var publicEventKinds = []string{"created", "status", "title"}

func addEvent(ctx context.Context, q querier, ticketID, actorID int64, kind, oldValue, newValue string) error {
	_, err := q.Exec(ctx, `
		INSERT INTO ticket_events (ticket_id, actor_id, kind, old_value, new_value)
		VALUES ($1, NULLIF($2, 0), $3, $4, $5)`, ticketID, actorID, kind, oldValue, newValue)
	return err
}

// recordChanges guarda en el historial cada campo que cambió entre before y after.
func recordChanges(ctx context.Context, q querier, actorID int64, before, after Ticket) error {
	type change struct{ kind, old, new string }
	var changes []change
	if before.Status != after.Status {
		changes = append(changes, change{"status", before.Status, after.Status})
	}
	if before.Priority != after.Priority {
		changes = append(changes, change{"priority", before.Priority, after.Priority})
	}
	if refID(before.Assignee) != refID(after.Assignee) {
		changes = append(changes, change{"assignee", refName(before.Assignee), refName(after.Assignee)})
	}
	if before.Category != after.Category {
		changes = append(changes, change{"category", before.Category, after.Category})
	}
	if before.Title != after.Title {
		changes = append(changes, change{"title", before.Title, after.Title})
	}
	if !slices.Equal(before.Tags, after.Tags) {
		changes = append(changes, change{"tags", strings.Join(before.Tags, ","), strings.Join(after.Tags, ",")})
	}
	for _, c := range changes {
		if err := addEvent(ctx, q, after.ID, actorID, c.kind, c.old, c.new); err != nil {
			return err
		}
	}
	return nil
}

func refID(u *UserRef) int64 {
	if u == nil {
		return 0
	}
	return u.ID
}

func refName(u *UserRef) string {
	if u == nil {
		return ""
	}
	return u.Name
}

// listEvents devuelve el historial en orden cronológico.
func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := s.loadTicket(w, r, id); !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), `
		SELECT e.id, u.id, u.name, e.kind, e.old_value, e.new_value, e.created_at
		FROM ticket_events e
		LEFT JOIN users u ON u.id = e.actor_id
		WHERE e.ticket_id = $1 AND ($2 OR e.kind = ANY($3))
		ORDER BY e.id`, id, claimsFrom(r).IsStaff(), publicEventKinds)
	if err != nil {
		internalError(w, err)
		return
	}
	events, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (TicketEvent, error) {
		var e TicketEvent
		var actorID *int64
		var actorName *string
		err := row.Scan(&e.ID, &actorID, &actorName, &e.Kind, &e.OldValue, &e.NewValue, &e.CreatedAt)
		if actorID != nil {
			e.Actor = &UserRef{ID: *actorID, Name: *actorName}
		}
		return e, err
	})
	if err != nil {
		internalError(w, err)
		return
	}
	if events == nil {
		events = []TicketEvent{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": events})
}
