package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// fanout reúne a quién avisar en tiempo real cuando la transacción termine bien.
type fanout struct {
	users   []int64
	tickets []ticketChange
}

type ticketChange struct {
	org       int64
	id        int64
	requester int64
	// staffOnly: el cambio no lo debe notar el cliente (por ejemplo, una nota interna).
	staffOnly bool
}

func (f *fanout) ticket(t Ticket, staffOnly bool) {
	f.tickets = append(f.tickets, ticketChange{org: t.OrgID, id: t.ID, requester: t.Requester.ID, staffOnly: staffOnly})
}

// notification es un aviso para un usuario: en la campana y, si lo quiere, por correo.
type notification struct {
	userID  int64
	actorID int64
	ticket  Ticket
	kind    string // new_ticket, assigned, comment, mention, status
	summary string
	// Cuerpo del correo; el enlace al ticket se añade al final.
	emailSubject string
	emailBody    string
}

// notify guarda la notificación y encola el correo dentro de la misma transacción.
// No se avisa a alguien de lo que hizo él mismo, ni a cuentas desactivadas.
func (s *Server) notify(ctx context.Context, q querier, out *fanout, n notification) error {
	if n.userID == n.actorID || n.userID == 0 {
		return nil
	}
	tag, err := q.Exec(ctx, `
		INSERT INTO notifications (user_id, ticket_id, actor_id, kind, summary)
		SELECT id, $2, NULLIF($3, 0), $4, $5 FROM users WHERE id = $1 AND active`,
		n.userID, n.ticket.ID, n.actorID, n.kind, n.summary)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	out.users = append(out.users, n.userID)

	body := n.emailBody + "\n\nVer el ticket: " + s.ticketURL(n.ticket.ID) +
		"\n\n—\nBeHIve. Puedes desactivar estos correos en tu perfil."
	_, err = q.Exec(ctx, `
		INSERT INTO email_outbox (to_email, subject, body)
		SELECT email, $2, $3 FROM users WHERE id = $1 AND active AND email_notifications`,
		n.userID, fmt.Sprintf("[#%d] %s", n.ticket.ID, n.emailSubject), body)
	return err
}

func (s *Server) ticketURL(id int64) string {
	return s.appURL + "/tickets/" + strconv.FormatInt(id, 10)
}

// excerpt recorta un texto largo para el correo.
func excerpt(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:max])) + "…"
}

func (s *Server) actorName(ctx context.Context, q querier, id int64) string {
	var name string
	if err := q.QueryRow(ctx, `SELECT name FROM users WHERE id = $1`, id).Scan(&name); err != nil {
		return "Alguien"
	}
	return name
}

// notifyNewTicket avisa a todo el equipo, y al cliente si un agente abrió el ticket en su nombre.
func (s *Server) notifyNewTicket(ctx context.Context, q querier, out *fanout, actorID int64, t Ticket) error {
	rows, err := q.Query(ctx, `SELECT id FROM users WHERE org_id = $1 AND active AND role IN ('agent', 'admin')`, t.OrgID)
	if err != nil {
		return err
	}
	var staff []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		staff = append(staff, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range staff {
		if err := s.notify(ctx, q, out, notification{
			userID: id, actorID: actorID, ticket: t, kind: "new_ticket",
			summary:      t.Requester.Name + " abrió un ticket nuevo",
			emailSubject: "Nuevo ticket: " + t.Title,
			emailBody: fmt.Sprintf("%s abrió un ticket nuevo (prioridad %s):\n\n%s\n\n%s",
				t.Requester.Name, priorityText[t.Priority], t.Title, excerpt(t.Description, 600)),
		}); err != nil {
			return err
		}
	}
	if actorID != t.Requester.ID {
		return s.notify(ctx, q, out, notification{
			userID: t.Requester.ID, actorID: actorID, ticket: t, kind: "new_ticket",
			summary:      "Se abrió un ticket a tu nombre",
			emailSubject: "Recibimos tu solicitud: " + t.Title,
			emailBody:    "El equipo de soporte abrió este ticket a tu nombre:\n\n" + t.Title,
		})
	}
	return nil
}

var statusText = map[string]string{
	"open": "abierto", "in_progress": "en curso", "waiting": "en espera de tu respuesta",
	"resolved": "resuelto", "closed": "cerrado",
}

var priorityText = map[string]string{"low": "baja", "medium": "media", "high": "alta", "urgent": "urgente"}

// notifyChanges avisa de una asignación nueva al agente y de los cambios de estado al cliente.
func (s *Server) notifyChanges(ctx context.Context, q querier, out *fanout, actorID int64, before, after Ticket) error {
	actor := s.actorName(ctx, q, actorID)
	if after.Assignee != nil && refID(before.Assignee) != after.Assignee.ID {
		if err := s.notify(ctx, q, out, notification{
			userID: after.Assignee.ID, actorID: actorID, ticket: after, kind: "assigned",
			summary:      actor + " te asignó un ticket",
			emailSubject: "Te asignaron: " + after.Title,
			emailBody:    fmt.Sprintf("%s te asignó este ticket:\n\n%s", actor, after.Title),
		}); err != nil {
			return err
		}
	}
	if before.Status != after.Status && actorID != after.Requester.ID &&
		(after.Status == "waiting" || after.Status == "resolved" || after.Status == "closed") {
		body := fmt.Sprintf("Tu ticket «%s» ahora está %s.", after.Title, statusText[after.Status])
		if after.Status == "resolved" {
			body += "\n\n¿Cómo te atendimos? Puedes valorarlo desde el ticket."
		}
		if err := s.notify(ctx, q, out, notification{
			userID: after.Requester.ID, actorID: actorID, ticket: after, kind: "status",
			summary:      "Tu ticket está " + statusText[after.Status],
			emailSubject: "Tu ticket está " + statusText[after.Status],
			emailBody:    body,
		}); err != nil {
			return err
		}
	}
	return nil
}

// notifyComment avisa a los mencionados, al cliente (respuestas públicas del equipo)
// y al agente asignado (mensajes del cliente o notas de compañeros).
func (s *Server) notifyComment(ctx context.Context, q querier, out *fanout, author actorInfo, t Ticket, c Comment) error {
	notified := map[int64]bool{author.id: true}
	send := func(userID int64, kind, summary, subject, intro string) error {
		if notified[userID] {
			return nil
		}
		notified[userID] = true
		return s.notify(ctx, q, out, notification{
			userID: userID, actorID: author.id, ticket: t, kind: kind, summary: summary,
			emailSubject: subject,
			emailBody:    intro + "\n\n" + excerpt(c.Body, 2000),
		})
	}

	if author.staff {
		mentioned, err := mentionedStaff(ctx, q, t.OrgID, c.Body)
		if err != nil {
			return err
		}
		for _, id := range mentioned {
			if err := send(id, "mention", author.name+" te mencionó",
				author.name+" te mencionó: "+t.Title, author.name+" te mencionó en un ticket:"); err != nil {
				return err
			}
		}
	}
	if author.staff && !c.Internal {
		if err := send(t.Requester.ID, "comment", author.name+" respondió a tu ticket",
			"Nueva respuesta: "+t.Title, author.name+" respondió:"); err != nil {
			return err
		}
	}
	if t.Assignee != nil {
		summary, intro := author.name+" respondió", author.name+" respondió:"
		if c.Internal {
			summary, intro = author.name+" añadió una nota interna", author.name+" añadió una nota interna:"
		}
		if err := send(t.Assignee.ID, "comment", summary, "Nuevo mensaje: "+t.Title, intro); err != nil {
			return err
		}
	}
	return nil
}

type actorInfo struct {
	id    int64
	name  string
	staff bool
}

// mentionedStaff busca "@Nombre" de agentes y administradores activos en el texto.
// Se comparan nombres completos para no confundir a "@Ana" con "@Ana María".
func mentionedStaff(ctx context.Context, q querier, orgID int64, body string) ([]int64, error) {
	if !strings.Contains(body, "@") {
		return nil, nil
	}
	rows, err := q.Query(ctx, `
		SELECT id, name FROM users WHERE org_id = $1 AND active AND role IN ('agent', 'admin')
		ORDER BY length(name) DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	text := strings.ToLower(body)
	var ids []int64
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		if containsMention(text, "@"+strings.ToLower(name)) {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

// containsMention exige que la mención no siga con otra letra: "@ana" no coincide en "@anabel".
func containsMention(text, mention string) bool {
	for start := 0; ; {
		i := strings.Index(text[start:], mention)
		if i < 0 {
			return false
		}
		end := start + i + len(mention)
		if end == len(text) {
			return true
		}
		r, _ := utf8.DecodeRuneInString(text[end:])
		if !isWordRune(r) {
			return true
		}
		start = end
	}
}

func isWordRune(r rune) bool {
	return r == '_' || r == '-' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
