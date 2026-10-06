package api_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
)

func (e *env) events(a actor, id int64) []api.TicketEvent {
	e.t.Helper()
	res := e.get(ticketPath(id)+"/events", a.Token)
	expectStatus(e.t, res, http.StatusOK)
	var out struct {
		Items []api.TicketEvent `json:"items"`
	}
	res.decode(&out)
	return out.Items
}

func kinds(events []api.TicketEvent) []string {
	out := make([]string, len(events))
	for i, ev := range events {
		out[i] = ev.Kind
	}
	return out
}

func TestTicketHistory(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	tk := e.newTicket(ana, map[string]any{"title": "Impresora"})

	expectStatus(t, e.patch(ticketPath(tk.ID), luis.Token, map[string]any{
		"status": "waiting", "priority": "high", "assignee_id": luis.ID, "tags": []string{"Hardware"},
	}), http.StatusOK)
	// Un PATCH sin cambios reales no ensucia el historial.
	expectStatus(t, e.patch(ticketPath(tk.ID), luis.Token, map[string]any{"priority": "high"}), http.StatusOK)
	// El cliente responde: el ticket vuelve a abierto y queda registrado.
	expectStatus(t, e.post(ticketPath(tk.ID)+"/comments", ana.Token, map[string]any{"body": "Sigue igual"}), http.StatusCreated)

	staff := e.events(luis, tk.ID)
	want := []string{"created", "status", "priority", "assignee", "tags", "status"}
	if !slices.Equal(kinds(staff), want) {
		t.Fatalf("historial = %v, se esperaba %v", kinds(staff), want)
	}
	if ev := staff[1]; ev.OldValue != "open" || ev.NewValue != "waiting" || ev.Actor == nil || ev.Actor.ID != luis.ID {
		t.Errorf("evento de estado = %+v", ev)
	}
	if ev := staff[3]; ev.OldValue != "" || ev.NewValue != "luis" {
		t.Errorf("evento de asignación = %+v", ev)
	}
	if ev := staff[5]; ev.NewValue != "open" || ev.Actor.ID != ana.ID {
		t.Errorf("reapertura = %+v", ev)
	}

	t.Run("el cliente solo ve creación, estados y título", func(t *testing.T) {
		got := kinds(e.events(ana, tk.ID))
		if !slices.Equal(got, []string{"created", "status", "status"}) {
			t.Errorf("historial del cliente = %v", got)
		}
	})
	t.Run("otro cliente no lo ve", func(t *testing.T) {
		otro := e.customer("otro")
		expectStatus(t, e.get(ticketPath(tk.ID)+"/events", otro.Token), http.StatusNotFound)
	})
}

func TestTags(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	a := e.newTicket(ana, map[string]any{"title": "A"})
	b := e.newTicket(ana, map[string]any{"title": "B"})

	res := e.patch(ticketPath(a.ID), luis.Token, map[string]any{"tags": []string{" Cliente VIP ", "cliente-vip", "facturación"}})
	expectStatus(t, res, http.StatusOK)
	var got api.Ticket
	res.decode(&got)
	if !slices.Equal(got.Tags, []string{"cliente-vip", "facturación"}) {
		t.Errorf("etiquetas normalizadas = %v", got.Tags)
	}
	expectStatus(t, e.patch(ticketPath(b.ID), luis.Token, map[string]any{"tags": []string{"facturación"}}), http.StatusOK)

	t.Run("filtrar por etiqueta", func(t *testing.T) {
		if got := ids(e.list(luis, "?tag=Cliente%20VIP").Items); !slices.Equal(got, []int64{a.ID}) {
			t.Errorf("tag=cliente-vip → %v", got)
		}
		if got := ids(e.list(luis, "?tag=facturación").Items); !slices.Equal(got, []int64{b.ID, a.ID}) {
			t.Errorf("tag=facturación → %v", got)
		}
	})

	t.Run("conteo de etiquetas", func(t *testing.T) {
		var out struct {
			Items []struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
			} `json:"items"`
		}
		e.get("/api/tags", luis.Token).decode(&out)
		if len(out.Items) != 2 || out.Items[0].Name != "facturación" || out.Items[0].Count != 2 {
			t.Errorf("tags = %+v", out.Items)
		}
		expectStatus(t, e.get("/api/tags", ana.Token), http.StatusForbidden)
	})

	t.Run("límites", func(t *testing.T) {
		many := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
		expectFieldError(t, e.patch(ticketPath(a.ID), luis.Token, map[string]any{"tags": many}), "tags")
		expectFieldError(t, e.patch(ticketPath(a.ID), luis.Token, map[string]any{"tags": []string{"una-etiqueta-larguísima-de-más-de-treinta"}}), "tags")
		expectFieldError(t, e.patch(ticketPath(a.ID), ana.Token, map[string]any{"tags": []string{"x"}}), "tags")
	})
}

func TestBulkUpdate(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	a := e.newTicket(ana, map[string]any{"title": "A"})
	b := e.newTicket(ana, map[string]any{"title": "B"})
	c := e.newTicket(ana, map[string]any{"title": "C"})
	expectStatus(t, e.patch(ticketPath(b.ID), luis.Token, map[string]any{"tags": []string{"red"}}), http.StatusOK)

	res := e.post("/api/tickets/bulk", luis.Token, map[string]any{
		"ids": []int64{a.ID, b.ID, b.ID, 999999}, "status": "in_progress", "assignee_id": luis.ID, "add_tags": []string{"Lote"},
	})
	expectStatus(t, res, http.StatusOK)
	if res.field("updated") != float64(2) {
		t.Errorf("updated = %v", res.field("updated"))
	}
	for _, id := range []int64{a.ID, b.ID} {
		got := e.ticket(luis, id)
		if got.Status != "in_progress" || got.Assignee == nil || !slices.Contains(got.Tags, "lote") {
			t.Errorf("ticket %d = %+v", id, got)
		}
	}
	if got := e.ticket(luis, b.ID).Tags; !slices.Equal(got, []string{"red", "lote"}) {
		t.Errorf("add_tags debe conservar las anteriores: %v", got)
	}
	if got := e.ticket(luis, c.ID); got.Status != "open" {
		t.Errorf("c no debía cambiar: %+v", got)
	}
	if got := kinds(e.events(luis, a.ID)); !slices.Equal(got, []string{"created", "status", "assignee", "tags"}) {
		t.Errorf("historial tras el lote = %v", got)
	}

	t.Run("solo agentes", func(t *testing.T) {
		expectStatus(t, e.post("/api/tickets/bulk", ana.Token, map[string]any{"ids": []int64{a.ID}, "status": "closed"}), http.StatusForbidden)
	})
	t.Run("validación", func(t *testing.T) {
		expectFieldError(t, e.post("/api/tickets/bulk", luis.Token, map[string]any{"ids": []int64{}, "status": "closed"}), "ids")
		expectFieldError(t, e.post("/api/tickets/bulk", luis.Token, map[string]any{"ids": []int64{a.ID}, "status": "nuevo"}), "status")
		big := make([]int64, 101)
		expectFieldError(t, e.post("/api/tickets/bulk", luis.Token, map[string]any{"ids": big, "status": "closed"}), "ids")
	})
}

func TestCategories(t *testing.T) {
	e := newEnv(t)
	boss := e.admin("jefa")
	luis := e.agent("luis")
	ana := e.customer("ana")

	res := e.post("/api/categories", boss.Token, map[string]any{"name": "Hardware"})
	expectStatus(t, res, http.StatusCreated)
	hwID := int64(res.field("id").(float64))
	expectFieldError(t, e.post("/api/categories", boss.Token, map[string]any{"name": "hardware"}), "name")
	expectStatus(t, e.post("/api/categories", luis.Token, map[string]any{"name": "Red"}), http.StatusForbidden)

	t.Run("el ticket usa el nombre canónico y rechaza categorías inexistentes", func(t *testing.T) {
		tk := e.newTicket(ana, map[string]any{"title": "x", "category": "HARDWARE"})
		if tk.Category != "Hardware" {
			t.Errorf("category = %q", tk.Category)
		}
		expectFieldError(t, e.post("/api/tickets", ana.Token, map[string]any{"title": "x", "category": "Cocina"}), "category")
		expectFieldError(t, e.patch(ticketPath(tk.ID), luis.Token, map[string]any{"category": "Cocina"}), "category")
	})

	t.Run("renombrar actualiza los tickets", func(t *testing.T) {
		tk := e.newTicket(ana, map[string]any{"title": "y", "category": "Hardware"})
		expectStatus(t, e.patch("/api/categories/"+itoa(hwID), boss.Token, map[string]any{"name": "Equipos"}), http.StatusOK)
		if got := e.ticket(ana, tk.ID).Category; got != "Equipos" {
			t.Errorf("category tras renombrar = %q", got)
		}
	})

	t.Run("listar y borrar", func(t *testing.T) {
		var out struct {
			Items []api.Category `json:"items"`
		}
		e.get("/api/categories", ana.Token).decode(&out)
		if len(out.Items) != 1 || out.Items[0].Name != "Equipos" {
			t.Errorf("categorías = %+v", out.Items)
		}
		expectStatus(t, e.do(http.MethodDelete, "/api/categories/"+itoa(hwID), boss.Token, nil), http.StatusNoContent)
		expectStatus(t, e.do(http.MethodDelete, "/api/categories/"+itoa(hwID), boss.Token, nil), http.StatusNotFound)
	})
}

func TestMacros(t *testing.T) {
	e := newEnv(t)
	luis := e.agent("luis")
	ana := e.customer("ana")

	res := e.post("/api/macros", luis.Token, map[string]any{
		"title": "Reiniciar", "body": "Hola {{solicitante}}, reinicia el equipo.", "status": "waiting",
	})
	expectStatus(t, res, http.StatusCreated)
	var m api.Macro
	res.decode(&m)
	if m.Status != "waiting" || m.Internal {
		t.Errorf("macro = %+v", m)
	}

	expectFieldError(t, e.post("/api/macros", luis.Token, map[string]any{"title": "x", "body": " "}), "body")
	expectFieldError(t, e.post("/api/macros", luis.Token, map[string]any{"title": "x", "body": "y", "status": "raro"}), "status")
	expectStatus(t, e.get("/api/macros", ana.Token), http.StatusForbidden)

	res = e.patch("/api/macros/"+itoa(m.ID), luis.Token, map[string]any{"title": "Reiniciar equipo", "body": "Texto", "internal": true})
	expectStatus(t, res, http.StatusOK)
	res.decode(&m)
	if m.Title != "Reiniciar equipo" || !m.Internal || m.Status != "" {
		t.Errorf("macro editada = %+v", m)
	}

	var out struct {
		Items []api.Macro `json:"items"`
	}
	e.get("/api/macros", luis.Token).decode(&out)
	if len(out.Items) != 1 {
		t.Errorf("macros = %+v", out.Items)
	}
	expectStatus(t, e.do(http.MethodDelete, "/api/macros/"+itoa(m.ID), luis.Token, nil), http.StatusNoContent)
	expectStatus(t, e.patch("/api/macros/"+itoa(m.ID), luis.Token, map[string]any{"title": "x", "body": "y"}), http.StatusNotFound)
}
