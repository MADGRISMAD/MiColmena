package api_test

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
)

func TestCreateTicketDefaults(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")

	tk := e.newTicket(ana, map[string]any{"title": "  No puedo entrar  "})

	if tk.Title != "No puedo entrar" {
		t.Errorf("el título debería recortarse: %q", tk.Title)
	}
	if tk.Status != "open" || tk.Priority != "medium" {
		t.Errorf("valores por defecto: status=%s priority=%s", tk.Status, tk.Priority)
	}
	if tk.Requester.ID != ana.ID || tk.Assignee != nil || tk.ResolvedAt != nil {
		t.Errorf("solicitante/asignado/resuelto inesperados: %+v", tk)
	}
	if string(tk.CustomFields) != "{}" {
		t.Errorf("custom_fields por defecto = %s", tk.CustomFields)
	}
}

func TestCreateTicketValidation(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")

	tests := []struct {
		name  string
		body  any
		field string // vacío = se espera 400 en vez de 422
	}{
		{"título vacío", map[string]any{"title": "   "}, "title"},
		{"título demasiado largo", map[string]any{"title": strings.Repeat("a", 201)}, "title"},
		{"descripción demasiado larga", map[string]any{"title": "x", "description": strings.Repeat("a", 20001)}, "description"},
		{"prioridad inválida", map[string]any{"title": "x", "priority": "altísima"}, "priority"},
		{"categoría demasiado larga", map[string]any{"title": "x", "category": strings.Repeat("a", 101)}, "category"},
		{"custom_fields no es un objeto", map[string]any{"title": "x", "custom_fields": []int{1}}, "custom_fields"},
		{"un cliente no puede crear en nombre de otro", map[string]any{"title": "x", "requester_id": luis.ID}, "requester_id"},
		{"campo desconocido", `{"title":"x","status":"closed"}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := e.post("/api/tickets", ana.Token, tt.body)
			if tt.field == "" {
				expectStatus(t, res, http.StatusBadRequest)
			} else {
				expectFieldError(t, res, tt.field)
			}
		})
	}

	t.Run("un agente puede crear en nombre de un cliente", func(t *testing.T) {
		tk := e.newTicket(luis, map[string]any{"title": "Llamada telefónica", "requester_id": ana.ID})
		if tk.Requester.ID != ana.ID {
			t.Errorf("solicitante = %d, se esperaba %d", tk.Requester.ID, ana.ID)
		}
		// Y la cliente lo ve en sus tickets.
		if got := ids(e.list(ana, "").Items); !slices.Contains(got, tk.ID) {
			t.Errorf("la cliente no ve el ticket creado para ella: %v", got)
		}
	})

	t.Run("solicitante que no existe", func(t *testing.T) {
		res := e.post("/api/tickets", luis.Token, map[string]any{"title": "x", "requester_id": 999999})
		expectFieldError(t, res, "requester_id")
	})

	t.Run("campos personalizados", func(t *testing.T) {
		tk := e.newTicket(ana, map[string]any{"title": "x", "custom_fields": map[string]any{"modelo": "HP 4100", "unidades": 3}})
		got := e.ticket(ana, tk.ID)
		if !strings.Contains(string(got.CustomFields), `"modelo"`) || !strings.Contains(string(got.CustomFields), "HP 4100") {
			t.Errorf("custom_fields = %s", got.CustomFields)
		}
	})
}

func TestCustomersOnlySeeTheirOwnTickets(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	beto := e.customer("beto")
	luis := e.agent("luis")

	a1 := e.newTicket(ana, map[string]any{"title": "De Ana 1"})
	a2 := e.newTicket(ana, map[string]any{"title": "De Ana 2"})
	b1 := e.newTicket(beto, map[string]any{"title": "De Beto"})

	if got := ids(e.list(ana, "").Items); !slices.Equal(got, []int64{a2.ID, a1.ID}) {
		t.Errorf("tickets de Ana = %v", got)
	}
	if got := ids(e.list(beto, "").Items); !slices.Equal(got, []int64{b1.ID}) {
		t.Errorf("tickets de Beto = %v", got)
	}
	if got := ids(e.list(luis, "").Items); len(got) != 3 {
		t.Errorf("un agente ve todos los tickets, vio %v", got)
	}

	t.Run("un ticket ajeno responde igual que uno que no existe", func(t *testing.T) {
		foreign := e.get(ticketPath(b1.ID), ana.Token)
		missing := e.get(ticketPath(999999), ana.Token)
		expectStatus(t, foreign, http.StatusNotFound)
		expectStatus(t, missing, http.StatusNotFound)
		if foreign.errorText() != missing.errorText() {
			t.Error("los mensajes distintos revelarían qué tickets existen")
		}
	})

	t.Run("ni leer, ni editar, ni comentar tickets ajenos", func(t *testing.T) {
		expectStatus(t, e.patch(ticketPath(b1.ID), ana.Token, map[string]any{"title": "hackeado"}), http.StatusNotFound)
		expectStatus(t, e.get(ticketPath(b1.ID)+"/comments", ana.Token), http.StatusNotFound)
		expectStatus(t, e.post(ticketPath(b1.ID)+"/comments", ana.Token, map[string]any{"body": "hola"}), http.StatusNotFound)
		if e.ticket(beto, b1.ID).Title != "De Beto" {
			t.Error("el ticket de Beto cambió")
		}
	})

	t.Run("id inválido", func(t *testing.T) {
		expectStatus(t, e.get("/api/tickets/abc", ana.Token), http.StatusBadRequest)
		expectStatus(t, e.get("/api/tickets/0", ana.Token), http.StatusBadRequest)
	})
}

func TestCustomerCannotChangeStaffFields(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	tk := e.newTicket(ana, map[string]any{"title": "Mío"})

	for _, field := range []struct {
		name  string
		value any
	}{
		{"priority", "urgent"},
		{"category", "VIP"},
		{"custom_fields", map[string]any{"a": 1}},
		{"assignee_id", luis.ID},
	} {
		res := e.patch(ticketPath(tk.ID), ana.Token, map[string]any{field.name: field.value})
		expectFieldError(t, res, field.name)
	}

	t.Run("solo puede cerrar, no cambiar a otros estados", func(t *testing.T) {
		expectFieldError(t, e.patch(ticketPath(tk.ID), ana.Token, map[string]any{"status": "in_progress"}), "status")
		expectFieldError(t, e.patch(ticketPath(tk.ID), ana.Token, map[string]any{"status": "resolved"}), "status")
	})

	t.Run("puede editar título y descripción", func(t *testing.T) {
		res := e.patch(ticketPath(tk.ID), ana.Token, map[string]any{"title": "Nuevo título", "description": "Más detalles"})
		expectStatus(t, res, http.StatusOK)
		got := e.ticket(ana, tk.ID)
		if got.Title != "Nuevo título" || got.Description != "Más detalles" {
			t.Errorf("no se guardó: %+v", got)
		}
	})

	t.Run("puede cerrar su propio ticket", func(t *testing.T) {
		res := e.patch(ticketPath(tk.ID), ana.Token, map[string]any{"status": "closed"})
		expectStatus(t, res, http.StatusOK)
		got := e.ticket(ana, tk.ID)
		if got.Status != "closed" || got.ResolvedAt == nil {
			t.Errorf("cerrar debería fijar resolved_at: %+v", got)
		}
	})

	t.Run("nada que cambiar devuelve el ticket tal cual", func(t *testing.T) {
		expectStatus(t, e.patch(ticketPath(tk.ID), ana.Token, map[string]any{}), http.StatusOK)
	})
}

func TestAssignTicket(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	root := e.admin("root")
	tk := e.newTicket(ana, map[string]any{})
	path := ticketPath(tk.ID)

	t.Run("a un agente", func(t *testing.T) {
		expectStatus(t, e.patch(path, luis.Token, map[string]any{"assignee_id": luis.ID}), http.StatusOK)
		got := e.ticket(luis, tk.ID)
		if got.Assignee == nil || got.Assignee.ID != luis.ID {
			t.Errorf("asignado = %+v", got.Assignee)
		}
	})

	t.Run("a un administrador", func(t *testing.T) {
		expectStatus(t, e.patch(path, luis.Token, map[string]any{"assignee_id": root.ID}), http.StatusOK)
	})

	t.Run("no a un cliente", func(t *testing.T) {
		expectFieldError(t, e.patch(path, luis.Token, map[string]any{"assignee_id": ana.ID}), "assignee_id")
	})

	t.Run("no a un usuario que no existe", func(t *testing.T) {
		expectFieldError(t, e.patch(path, luis.Token, map[string]any{"assignee_id": 999999}), "assignee_id")
	})

	t.Run("valor que no es un número", func(t *testing.T) {
		expectFieldError(t, e.patch(path, luis.Token, `{"assignee_id":"luis"}`), "assignee_id")
	})

	t.Run("null desasigna", func(t *testing.T) {
		expectStatus(t, e.patch(path, luis.Token, `{"assignee_id":null}`), http.StatusOK)
		if got := e.ticket(luis, tk.ID); got.Assignee != nil {
			t.Errorf("debería quedar sin asignar: %+v", got.Assignee)
		}
	})
}

func TestResolvedAtFollowsStatus(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	tk := e.newTicket(ana, map[string]any{})
	set := func(status string) api.Ticket {
		t.Helper()
		expectStatus(t, e.patch(ticketPath(tk.ID), luis.Token, map[string]any{"status": status}), http.StatusOK)
		return e.ticket(luis, tk.ID)
	}

	if tk.ResolvedAt != nil {
		t.Fatal("un ticket nuevo no tiene fecha de resolución")
	}
	for _, s := range []string{"in_progress", "waiting"} {
		if got := set(s); got.ResolvedAt != nil {
			t.Errorf("en %s no debería haber fecha de resolución", s)
		}
	}

	resolved := set("resolved")
	if resolved.ResolvedAt == nil {
		t.Fatal("al resolver se debe fijar resolved_at")
	}

	// Pasar de resuelto a cerrado no cambia cuándo se resolvió.
	closed := set("closed")
	if closed.ResolvedAt == nil || !closed.ResolvedAt.Equal(*resolved.ResolvedAt) {
		t.Errorf("resolved_at cambió al cerrar: %v -> %v", resolved.ResolvedAt, closed.ResolvedAt)
	}

	// Reabrir lo borra.
	if got := set("open"); got.ResolvedAt != nil {
		t.Errorf("al reabrir debe borrarse resolved_at, quedó %v", got.ResolvedAt)
	}

	// Y al volver a resolver se fija de nuevo.
	if got := set("resolved"); got.ResolvedAt == nil {
		t.Error("al resolver otra vez se debe fijar resolved_at")
	}
}

func TestUpdateBumpsUpdatedAt(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	tk := e.newTicket(ana, map[string]any{})

	time.Sleep(10 * time.Millisecond)
	expectStatus(t, e.patch(ticketPath(tk.ID), luis.Token, map[string]any{"priority": "high"}), http.StatusOK)

	got := e.ticket(luis, tk.ID)
	if !got.UpdatedAt.After(tk.UpdatedAt) {
		t.Errorf("updated_at no avanzó: %v -> %v", tk.UpdatedAt, got.UpdatedAt)
	}
	if !got.CreatedAt.Equal(tk.CreatedAt) {
		t.Error("created_at no debe cambiar")
	}
}

func TestUpdateValidation(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	path := ticketPath(e.newTicket(ana, map[string]any{}).ID)

	expectFieldError(t, e.patch(path, luis.Token, map[string]any{"status": "borrado"}), "status")
	expectFieldError(t, e.patch(path, luis.Token, map[string]any{"priority": "altísima"}), "priority")
	expectFieldError(t, e.patch(path, luis.Token, map[string]any{"title": "  "}), "title")
	expectFieldError(t, e.patch(path, luis.Token, map[string]any{"custom_fields": "texto"}), "custom_fields")
	expectStatus(t, e.patch(path, luis.Token, `{"requester_id": 1}`), http.StatusBadRequest) // no se puede cambiar el solicitante
	expectStatus(t, e.patch("/api/tickets/999999", luis.Token, map[string]any{"title": "x"}), http.StatusNotFound)
}

func TestListFilters(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	root := e.admin("root")

	impresora := e.newTicket(ana, map[string]any{"title": "La impresora no imprime", "priority": "high"})
	factura := e.newTicket(ana, map[string]any{"title": "Factura duplicada", "description": "Me cobraron dos veces el mes pasado"})
	acceso := e.newTicket(ana, map[string]any{"title": "No puedo acceder", "priority": "urgent"})

	expectStatus(t, e.patch(ticketPath(impresora.ID), luis.Token, map[string]any{"assignee_id": luis.ID, "status": "in_progress"}), http.StatusOK)
	expectStatus(t, e.patch(ticketPath(acceso.ID), luis.Token, map[string]any{"assignee_id": root.ID}), http.StatusOK)

	check := func(who actor, query string, want ...int64) {
		t.Helper()
		got := ids(e.list(who, query).Items)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("GET /api/tickets%s => %v, se esperaba %v", query, got, want)
		}
	}

	check(luis, "?status=in_progress", impresora.ID)
	check(luis, "?status=open", factura.ID, acceso.ID)
	check(luis, "?priority=urgent", acceso.ID)
	check(luis, "?assignee=me", impresora.ID)
	check(root, "?assignee=me", acceso.ID)
	check(luis, "?assignee=none", factura.ID)
	check(luis, "?assignee="+strconv.FormatInt(root.ID, 10), acceso.ID)
	check(luis, "?status=open&assignee=none", factura.ID)
	check(luis, "?status=open&priority=high") // ninguno cumple las dos

	t.Run("búsqueda de texto en español", func(t *testing.T) {
		check(luis, "?q=impresoras", impresora.ID) // plural encuentra el singular
		check(luis, "?q=cobraron", factura.ID)     // busca también en la descripción
		check(luis, "?q=cobrar", factura.ID)       // y entiende otras formas del verbo
		check(luis, "?q=factura+mes", factura.ID)
		check(luis, "?q=zzzzzz")
	})

	t.Run("los filtros no saltan los permisos", func(t *testing.T) {
		beto := e.customer("beto")
		e.newTicket(beto, map[string]any{"title": "Impresora de Beto"})
		check(ana, "?q=impresora", impresora.ID) // no aparece la de Beto
		check(ana, "?assignee=none", factura.ID)
	})

	t.Run("valores inválidos", func(t *testing.T) {
		for _, q := range []string{"?status=borrado", "?priority=x", "?assignee=abc", "?before=abc", "?limit=0", "?limit=101", "?limit=x"} {
			if res := e.get("/api/tickets"+q, luis.Token); res.Status != http.StatusBadRequest {
				t.Errorf("GET /api/tickets%s => %d, se esperaba 400", q, res.Status)
			}
		}
	})

	t.Run("lista vacía es [] y no null", func(t *testing.T) {
		res := e.get("/api/tickets?q=zzzzzz", luis.Token)
		if !strings.Contains(string(res.Body), `"items":[]`) {
			t.Errorf("respuesta: %s", res.Body)
		}
	})
}

func TestPaginationWithCursor(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	var created []int64
	for i := 0; i < 7; i++ {
		created = append(created, e.newTicket(ana, map[string]any{"title": "Ticket " + strconv.Itoa(i)}).ID)
	}
	slices.Reverse(created) // el más nuevo primero

	var seen []int64
	query := "?limit=3"
	for pages := 0; ; pages++ {
		if pages > 5 {
			t.Fatal("la paginación no termina")
		}
		page := e.list(ana, query)
		if len(page.Items) > 3 {
			t.Fatalf("la página trae %d tickets con limit=3", len(page.Items))
		}
		seen = append(seen, ids(page.Items)...)
		if page.NextCursor == nil {
			break
		}
		if last := page.Items[len(page.Items)-1].ID; *page.NextCursor != last {
			t.Errorf("next_cursor = %d, se esperaba el id del último ticket (%d)", *page.NextCursor, last)
		}
		query = "?limit=3&before=" + strconv.FormatInt(*page.NextCursor, 10)
	}

	if !slices.Equal(seen, created) {
		t.Errorf("recorrido = %v, se esperaba %v (sin repetir ni saltar tickets)", seen, created)
	}

	t.Run("una página exacta no inventa otra", func(t *testing.T) {
		page := e.list(ana, "?limit=7")
		if len(page.Items) != 7 || page.NextCursor != nil {
			t.Errorf("items=%d next=%v", len(page.Items), page.NextCursor)
		}
	})

	t.Run("nuevos tickets no desplazan las páginas", func(t *testing.T) {
		first := e.list(ana, "?limit=3")
		e.newTicket(ana, map[string]any{"title": "Llegó uno nuevo"})
		second := e.list(ana, "?limit=3&before="+strconv.FormatInt(*first.NextCursor, 10))
		for _, tk := range second.Items {
			if slices.Contains(ids(first.Items), tk.ID) {
				t.Errorf("el ticket %d se repite en la segunda página", tk.ID)
			}
		}
	})
}
