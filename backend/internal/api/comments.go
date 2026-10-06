package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Comment struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticket_id"`
	Author    UserRef   `json:"author"`
	Body      string    `json:"body"`
	Internal  bool      `json:"internal"`
	CreatedAt time.Time `json:"created_at"`
}

const commentSelect = `
	SELECT c.id, c.ticket_id, u.id, u.name, c.body, c.internal, c.created_at
	FROM ticket_comments c
	JOIN users u ON u.id = c.author_id`

func scanComment(row pgx.Row) (Comment, error) {
	var c Comment
	err := row.Scan(&c.ID, &c.TicketID, &c.Author.ID, &c.Author.Name, &c.Body, &c.Internal, &c.CreatedAt)
	return c, err
}

// listComments devuelve los comentarios en orden cronológico. Los clientes no ven las notas internas.
func (s *Server) listComments(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := s.loadTicket(w, r, id); !ok {
		return
	}

	rows, err := s.db.Query(r.Context(), commentSelect+`
		WHERE c.ticket_id = $1 AND ($2 OR NOT c.internal)
		ORDER BY c.id`, id, claimsFrom(r).IsStaff())
	if err != nil {
		internalError(w, err)
		return
	}
	comments, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Comment, error) { return scanComment(row) })
	if err != nil {
		internalError(w, err)
		return
	}
	if comments == nil {
		comments = []Comment{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": comments})
}

func (s *Server) createComment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	ticket, ok := s.loadTicket(w, r, id)
	if !ok {
		return
	}
	claims := claimsFrom(r)

	var in struct {
		Body     string `json:"body"`
		Internal bool   `json:"internal"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	errs := validationErrors{}
	errs.check(notBlank(in.Body), "body", "es obligatorio")
	errs.check(len(in.Body) <= 20000, "body", "es demasiado largo")
	errs.check(!in.Internal || claims.IsStaff(), "internal", "solo un agente puede crear notas internas")
	if errs.write(w) {
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())

	var commentID int64
	if err := tx.QueryRow(r.Context(), `
		INSERT INTO ticket_comments (ticket_id, author_id, body, internal)
		VALUES ($1, $2, $3, $4)
		RETURNING id`, id, claims.UserID(), strings.TrimSpace(in.Body), in.Internal,
	).Scan(&commentID); err != nil {
		internalError(w, err)
		return
	}

	// Si el cliente responde a un ticket que esperaba su respuesta, vuelve a quedar abierto.
	reopen := !claims.IsStaff() && ticket.Status == "waiting"
	// La primera respuesta pública de un agente cuenta para el SLA.
	firstResponse := claims.IsStaff() && !in.Internal && claims.UserID() != ticket.Requester.ID
	if _, err := tx.Exec(r.Context(), `
		UPDATE tickets
		SET updated_at = now(),
		    status = CASE WHEN $2 THEN 'open' ELSE status END,
		    first_response_at = CASE WHEN $3 THEN coalesce(first_response_at, now()) ELSE first_response_at END
		WHERE id = $1`, id, reopen, firstResponse); err != nil {
		internalError(w, err)
		return
	}
	if reopen {
		if err := addEvent(r.Context(), tx, id, claims.UserID(), "status", ticket.Status, "open"); err != nil {
			internalError(w, err)
			return
		}
	}

	c, err := scanComment(tx.QueryRow(r.Context(), commentSelect+" WHERE c.id = $1", commentID))
	if err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}
