package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type Notification struct {
	ID        int64      `json:"id"`
	TicketID  *int64     `json:"ticket_id"`
	Actor     *UserRef   `json:"actor"`
	Kind      string     `json:"kind"`
	Summary   string     `json:"summary"`
	Title     string     `json:"ticket_title"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// listNotifications devuelve las últimas notificaciones y cuántas faltan por leer.
func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	userID := claimsFrom(r).UserID()
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			writeError(w, http.StatusBadRequest, "limit debe estar entre 1 y 100")
			return
		}
		limit = n
	}
	rows, err := s.db.Query(r.Context(), `
		SELECT n.id, n.ticket_id, u.id, u.name, n.kind, n.summary, coalesce(t.title, ''), n.read_at, n.created_at
		FROM notifications n
		LEFT JOIN users u ON u.id = n.actor_id
		LEFT JOIN tickets t ON t.id = n.ticket_id
		WHERE n.user_id = $1
		ORDER BY n.id DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Notification, error) {
		var n Notification
		var actorID *int64
		var actorName *string
		err := row.Scan(&n.ID, &n.TicketID, &actorID, &actorName, &n.Kind, &n.Summary, &n.Title, &n.ReadAt, &n.CreatedAt)
		if actorID != nil {
			n.Actor = &UserRef{ID: *actorID, Name: *actorName}
		}
		return n, err
	})
	if err != nil {
		internalError(w, err)
		return
	}
	if items == nil {
		items = []Notification{}
	}
	var unread int
	if err := s.db.QueryRow(r.Context(),
		`SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID,
	).Scan(&unread); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "unread": unread})
}

// readNotifications marca como leídas las indicadas, las de un ticket, o todas si no se indica nada.
func (s *Server) readNotifications(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs      []int64 `json:"ids"`
		TicketID *int64  `json:"ticket_id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	_, err := s.db.Exec(r.Context(), `
		UPDATE notifications SET read_at = now()
		WHERE user_id = $1 AND read_at IS NULL
		  AND (coalesce(cardinality($2::bigint[]), 0) = 0 OR id = ANY($2))
		  AND ($3::bigint IS NULL OR ticket_id = $3)`,
		claimsFrom(r).UserID(), in.IDs, in.TicketID)
	if err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
