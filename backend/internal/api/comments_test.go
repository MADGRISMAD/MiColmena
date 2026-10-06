package api_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
)

type commentList struct {
	Items []api.Comment `json:"items"`
}

func (e *env) comments(a actor, ticketID int64) []api.Comment {
	e.t.Helper()
	res := e.get(ticketPath(ticketID)+"/comments", a.Token)
	if res.Status != http.StatusOK {
		e.t.Fatalf("listar comentarios: %d %s", res.Status, res.Body)
	}
	var l commentList
	res.decode(&l)
	return l.Items
}

func bodies(comments []api.Comment) []string {
	out := make([]string, len(comments))
	for i, c := range comments {
		out[i] = c.Body
	}
	return out
}

func TestInternalNotesAreHiddenFromCustomers(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	tk := e.newTicket(ana, map[string]any{})
	path := ticketPath(tk.ID) + "/comments"

	expectStatus(t, e.post(path, ana.Token, map[string]any{"body": "Pública de Ana"}), http.StatusCreated)
	expectStatus(t, e.post(path, luis.Token, map[string]any{"body": "Nota interna", "internal": true}), http.StatusCreated)
	expectStatus(t, e.post(path, luis.Token, map[string]any{"body": "Respuesta pública"}), http.StatusCreated)

	staff := e.comments(luis, tk.ID)
	if got := strings.Join(bodies(staff), "|"); got != "Pública de Ana|Nota interna|Respuesta pública" {
		t.Errorf("el agente ve %q", got)
	}
	customer := e.comments(ana, tk.ID)
	if got := strings.Join(bodies(customer), "|"); got != "Pública de Ana|Respuesta pública" {
		t.Errorf("la cliente ve %q, no debería ver la nota interna", got)
	}
	if staff[1].Internal != true || staff[0].Internal != false {
		t.Errorf("el campo internal no coincide: %+v", staff)
	}

	t.Run("la nota interna no se filtra en ninguna otra respuesta", func(t *testing.T) {
		for _, path := range []string{ticketPath(tk.ID), "/api/tickets"} {
			if res := e.get(path, ana.Token); strings.Contains(string(res.Body), "Nota interna") {
				t.Errorf("%s filtra la nota interna: %s", path, res.Body)
			}
		}
	})
}

func TestCustomerCannotCreateInternalNotes(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	tk := e.newTicket(ana, map[string]any{})

	res := e.post(ticketPath(tk.ID)+"/comments", ana.Token, map[string]any{"body": "intento", "internal": true})
	expectFieldError(t, res, "internal")
	if n := len(e.comments(ana, tk.ID)); n != 0 {
		t.Errorf("no debería haberse guardado nada, hay %d comentarios", n)
	}
}

func TestCommentValidation(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	tk := e.newTicket(ana, map[string]any{})
	path := ticketPath(tk.ID) + "/comments"

	expectFieldError(t, e.post(path, ana.Token, map[string]any{"body": "  \n "}), "body")
	expectFieldError(t, e.post(path, ana.Token, map[string]any{"body": strings.Repeat("a", 20001)}), "body")
	expectStatus(t, e.post(path, ana.Token, `{"body":"x","ticket_id":5}`), http.StatusBadRequest)
	expectStatus(t, e.post("/api/tickets/999999/comments", ana.Token, map[string]any{"body": "x"}), http.StatusNotFound)

	res := e.post(path, ana.Token, map[string]any{"body": "  con espacios  "})
	expectStatus(t, res, http.StatusCreated)
	var c api.Comment
	res.decode(&c)
	if c.Body != "con espacios" || c.Author.ID != ana.ID || c.TicketID != tk.ID {
		t.Errorf("comentario creado = %+v", c)
	}
}

func TestCommentsAreInChronologicalOrder(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	tk := e.newTicket(ana, map[string]any{})
	path := ticketPath(tk.ID) + "/comments"

	for _, b := range []string{"uno", "dos", "tres"} {
		expectStatus(t, e.post(path, luis.Token, map[string]any{"body": b}), http.StatusCreated)
	}
	if got := strings.Join(bodies(e.comments(ana, tk.ID)), ","); got != "uno,dos,tres" {
		t.Errorf("orden = %s", got)
	}
}

func TestCustomerReplyReopensWaitingTicket(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	tk := e.newTicket(ana, map[string]any{})
	path := ticketPath(tk.ID) + "/comments"

	expectStatus(t, e.patch(ticketPath(tk.ID), luis.Token, map[string]any{"status": "waiting"}), http.StatusOK)

	t.Run("la respuesta del agente no cambia el estado", func(t *testing.T) {
		expectStatus(t, e.post(path, luis.Token, map[string]any{"body": "¿Puedes enviarme una captura?"}), http.StatusCreated)
		if got := e.ticket(luis, tk.ID).Status; got != "waiting" {
			t.Errorf("estado = %s, debería seguir en espera", got)
		}
	})

	t.Run("una nota interna tampoco", func(t *testing.T) {
		expectStatus(t, e.post(path, luis.Token, map[string]any{"body": "x", "internal": true}), http.StatusCreated)
		if got := e.ticket(luis, tk.ID).Status; got != "waiting" {
			t.Errorf("estado = %s", got)
		}
	})

	t.Run("la respuesta de la cliente lo reabre", func(t *testing.T) {
		expectStatus(t, e.post(path, ana.Token, map[string]any{"body": "Aquí está"}), http.StatusCreated)
		if got := e.ticket(luis, tk.ID).Status; got != "open" {
			t.Errorf("estado = %s, debería haberse reabierto", got)
		}
	})

	t.Run("en otros estados no cambia nada", func(t *testing.T) {
		expectStatus(t, e.patch(ticketPath(tk.ID), luis.Token, map[string]any{"status": "in_progress"}), http.StatusOK)
		expectStatus(t, e.post(path, ana.Token, map[string]any{"body": "¿Novedades?"}), http.StatusCreated)
		if got := e.ticket(luis, tk.ID).Status; got != "in_progress" {
			t.Errorf("estado = %s", got)
		}
	})
}

func TestCommentBumpsTicketUpdatedAt(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	tk := e.newTicket(ana, map[string]any{})

	time.Sleep(10 * time.Millisecond)
	expectStatus(t, e.post(ticketPath(tk.ID)+"/comments", ana.Token, map[string]any{"body": "hola"}), http.StatusCreated)

	if got := e.ticket(ana, tk.ID); !got.UpdatedAt.After(tk.UpdatedAt) {
		t.Errorf("updated_at no avanzó: %v -> %v", tk.UpdatedAt, got.UpdatedAt)
	}
}

func TestStats(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")

	type stats struct {
		ByStatus   map[string]int `json:"by_status"`
		OpenByPrio map[string]int `json:"open_by_priority"`
		Unassigned int            `json:"unassigned_open"`
		AvgHours   *float64       `json:"avg_resolution_hours_30d"`
	}
	get := func() stats {
		t.Helper()
		res := e.get("/api/stats", luis.Token)
		expectStatus(t, res, http.StatusOK)
		var s stats
		res.decode(&s)
		return s
	}

	t.Run("sin tickets todo está en cero y sin promedio", func(t *testing.T) {
		s := get()
		for _, st := range []string{"open", "in_progress", "waiting", "resolved", "closed"} {
			if v, ok := s.ByStatus[st]; !ok || v != 0 {
				t.Errorf("by_status[%s] = %v (presente: %v)", st, v, ok)
			}
		}
		for _, p := range []string{"low", "medium", "high", "urgent"} {
			if v, ok := s.OpenByPrio[p]; !ok || v != 0 {
				t.Errorf("open_by_priority[%s] = %v (presente: %v)", p, v, ok)
			}
		}
		if s.Unassigned != 0 || s.AvgHours != nil {
			t.Errorf("unassigned=%d avg=%v", s.Unassigned, s.AvgHours)
		}
	})

	open := e.newTicket(ana, map[string]any{"priority": "high"})
	progress := e.newTicket(ana, map[string]any{"priority": "high"})
	urgent := e.newTicket(ana, map[string]any{"priority": "urgent"})
	solved := e.newTicket(ana, map[string]any{"priority": "low"})
	old := e.newTicket(ana, map[string]any{"priority": "low"})
	_ = open

	expectStatus(t, e.patch(ticketPath(progress.ID), luis.Token, map[string]any{"status": "in_progress", "assignee_id": luis.ID}), http.StatusOK)
	expectStatus(t, e.patch(ticketPath(solved.ID), luis.Token, map[string]any{"status": "resolved"}), http.StatusOK)
	expectStatus(t, e.patch(ticketPath(old.ID), luis.Token, map[string]any{"status": "closed"}), http.StatusOK)
	_ = urgent

	// Un ticket resuelto hace 40 días, tras 10 horas de trabajo, no cuenta para el promedio de 30 días.
	// El de hace 2 días, resuelto tras 4 horas, sí.
	mustExec(t, e, `UPDATE tickets SET created_at = now() - interval '40 days', resolved_at = now() - interval '40 days' + interval '10 hours' WHERE id = $1`, old.ID)
	mustExec(t, e, `UPDATE tickets SET created_at = now() - interval '2 days', resolved_at = now() - interval '2 days' + interval '4 hours' WHERE id = $1`, solved.ID)

	s := get()
	wantStatus := map[string]int{"open": 2, "in_progress": 1, "waiting": 0, "resolved": 1, "closed": 1}
	for st, want := range wantStatus {
		if s.ByStatus[st] != want {
			t.Errorf("by_status[%s] = %d, se esperaba %d", st, s.ByStatus[st], want)
		}
	}
	// Las prioridades solo cuentan los tickets sin resolver: 1 high abierto + 1 high en curso + 1 urgent.
	wantPrio := map[string]int{"high": 2, "urgent": 1, "medium": 0, "low": 0}
	for p, want := range wantPrio {
		if s.OpenByPrio[p] != want {
			t.Errorf("open_by_priority[%s] = %d, se esperaba %d (los resueltos y cerrados no cuentan)", p, s.OpenByPrio[p], want)
		}
	}
	// Sin asignar y sin resolver: el abierto y el urgente (el en curso sí tiene agente).
	if s.Unassigned != 2 {
		t.Errorf("unassigned_open = %d, se esperaba 2", s.Unassigned)
	}
	if s.AvgHours == nil || *s.AvgHours < 3.9 || *s.AvgHours > 4.1 {
		t.Errorf("promedio = %v, se esperaban 4 horas", s.AvgHours)
	}

	t.Run("solo para el equipo", func(t *testing.T) {
		expectStatus(t, e.get("/api/stats", ana.Token), http.StatusForbidden)
		expectStatus(t, e.get("/api/stats", ""), http.StatusUnauthorized)
	})
}
