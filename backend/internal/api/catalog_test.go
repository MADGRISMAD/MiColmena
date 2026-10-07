package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
)

type catalogResponse struct {
	Categories []api.ServiceCategory `json:"categories"`
}

func (e *env) catalog(token, query string) []api.ServiceCategory {
	e.t.Helper()
	res := e.get("/api/catalog"+query, token)
	expectStatus(e.t, res, http.StatusOK)
	var out catalogResponse
	res.decode(&out)
	return out.Categories
}

// firstItem busca un servicio del catálogo por su nombre.
func firstItem(t *testing.T, cats []api.ServiceCategory, name string) api.ServiceItem {
	t.Helper()
	for _, c := range cats {
		for _, it := range c.Items {
			if it.Name == name {
				return it
			}
		}
	}
	t.Fatalf("no hay un servicio %q", name)
	return api.ServiceItem{}
}

func TestServiceCatalogRequests(t *testing.T) {
	e := newEnv(t)
	acme := e.signup("Acme", "admin@acme.test", 20)
	ae := e.in(acme)
	ana := ae.customer("ana")

	cats := e.catalog(ana.Token, "")
	if len(cats) != 3 {
		t.Fatalf("una empresa nueva trae 3 categorías de ejemplo, hay %d", len(cats))
	}
	access := firstItem(t, cats, "Acceso a un sistema")

	t.Run("el formulario se valida", func(t *testing.T) {
		res := e.post("/api/catalog/items/"+itoa(access.ID)+"/request", ana.Token, map[string]any{"values": map[string]any{}})
		expectStatus(t, res, http.StatusUnprocessableEntity)
		for _, f := range []string{"values.sistema", "values.nivel"} {
			if _, ok := res.fieldErrors()[f]; !ok {
				t.Errorf("falta error en %s: %s", f, res.Body)
			}
		}
		res = e.post("/api/catalog/items/"+itoa(access.ID)+"/request", ana.Token,
			map[string]any{"values": map[string]any{"sistema": "ERP", "nivel": "Dios"}})
		expectStatus(t, res, http.StatusUnprocessableEntity)
		expectFieldError(t, res, "values.nivel")
	})

	var request api.Ticket
	t.Run("pedir un servicio crea una solicitud", func(t *testing.T) {
		res := e.post("/api/catalog/items/"+itoa(access.ID)+"/request", ana.Token, map[string]any{
			"values": map[string]any{"sistema": "ERP", "nivel": "Solo lectura", "motivo": "Voy a cerrar el mes"},
			"notes":  "Es urgente",
		})
		expectStatus(t, res, http.StatusCreated)
		res.decode(&request)
		if request.Kind != "request" || request.Title != "Acceso a un sistema" || request.Requester.ID != ana.ID ||
			request.CatalogItem == nil || request.CatalogItem.ID != access.ID {
			t.Errorf("ticket = %+v", request)
		}
		for _, want := range []string{"¿A qué sistema o carpeta?: ERP", "Nivel de acceso: Solo lectura", "Es urgente"} {
			if !strings.Contains(request.Description, want) {
				t.Errorf("la descripción no tiene %q: %q", want, request.Description)
			}
		}
		if !strings.Contains(string(request.CustomFields), `"sistema":"ERP"`) {
			t.Errorf("custom_fields = %s", request.CustomFields)
		}
	})

	t.Run("los avisos y los filtros distinguen solicitudes de incidentes", func(t *testing.T) {
		incident := ae.newTicket(ana, map[string]any{"title": "No imprime"})
		if incident.Kind != "incident" {
			t.Errorf("kind = %q", incident.Kind)
		}
		got := ae.list(acme.admin, "?kind=request")
		if len(got.Items) != 1 || got.Items[0].ID != request.ID {
			t.Errorf("solicitudes = %v", ids(got.Items))
		}
		got = ae.list(acme.admin, "?kind=incident")
		if len(got.Items) != 1 || got.Items[0].ID != incident.ID {
			t.Errorf("incidentes = %v", ids(got.Items))
		}
		expectStatus(t, e.get("/api/tickets?kind=otro", acme.admin.Token), http.StatusBadRequest)

		var stats struct {
			RequestsOpen int `json:"requests_open"`
			MineOpen     int `json:"mine_open"`
		}
		e.get("/api/stats", acme.admin.Token).decode(&stats)
		if stats.RequestsOpen != 1 || stats.MineOpen != 0 {
			t.Errorf("stats = %+v", stats)
		}
	})

	t.Run("solo los administradores editan el catálogo", func(t *testing.T) {
		agent := ae.agent("agente")
		body := map[string]any{"category_id": cats[0].ID, "name": "Nuevo"}
		expectStatus(t, e.post("/api/catalog/items", ana.Token, body), http.StatusForbidden)
		expectStatus(t, e.post("/api/catalog/items", agent.Token, body), http.StatusForbidden)
		expectStatus(t, e.post("/api/catalog/categories", agent.Token, map[string]any{"name": "X"}), http.StatusForbidden)
		if e.catalog(ana.Token, "?all=1")[0].Items == nil {
			t.Error("?all=1 no debe romper para un cliente")
		}
	})

	t.Run("el administrador crea y valida servicios", func(t *testing.T) {
		token := acme.admin.Token
		res := e.post("/api/catalog/items", token, map[string]any{
			"category_id": cats[0].ID, "name": "Licencia de software", "priority": "low",
			"fields": []map[string]any{
				{"label": "Programa", "type": "text", "required": true},
				{"label": "Versión", "type": "select", "options": []string{"2023", "2024"}},
			},
		})
		expectStatus(t, res, http.StatusCreated)
		var item api.ServiceItem
		res.decode(&item)
		if len(item.Fields) != 2 || item.Fields[0].Key == "" || item.Fields[0].Key == item.Fields[1].Key {
			t.Errorf("los campos reciben identificador propio: %+v", item.Fields)
		}

		bad := func(name string, body map[string]any) {
			t.Helper()
			body["category_id"], body["name"] = cats[0].ID, "Malo"
			res := e.post("/api/catalog/items", token, body)
			if res.Status != http.StatusUnprocessableEntity {
				t.Errorf("%s: %d %s", name, res.Status, res.Body)
			}
		}
		bad("tipo desconocido", map[string]any{"fields": []map[string]any{{"label": "A", "type": "video"}}})
		bad("lista sin opciones", map[string]any{"fields": []map[string]any{{"label": "A", "type": "select"}}})
		bad("campo sin nombre", map[string]any{"fields": []map[string]any{{"label": " ", "type": "text"}}})
		bad("prioridad", map[string]any{"priority": "ya"})
		bad("categoría de tickets", map[string]any{"ticket_category": "No existe"})
		bad("clave repetida", map[string]any{"fields": []map[string]any{
			{"key": "a", "label": "A", "type": "text"}, {"key": "a", "label": "B", "type": "text"}}})

		// Desactivar un servicio lo quita del catálogo y ya no se puede pedir.
		off := false
		res = e.patch("/api/catalog/items/"+itoa(item.ID), token, map[string]any{
			"category_id": cats[0].ID, "name": item.Name, "priority": "low", "fields": item.Fields, "active": off})
		expectStatus(t, res, http.StatusOK)
		if _, found := findItem(e.catalog(ana.Token, ""), item.ID); found {
			t.Error("un servicio inactivo no se muestra a los clientes")
		}
		if _, found := findItem(e.catalog(token, "?all=1"), item.ID); !found {
			t.Error("el administrador sí lo ve para editarlo")
		}
		expectStatus(t, e.post("/api/catalog/items/"+itoa(item.ID)+"/request", ana.Token,
			map[string]any{"values": map[string]any{"programa": "x"}}), http.StatusNotFound)
	})

	t.Run("categorías", func(t *testing.T) {
		token := acme.admin.Token
		expectStatus(t, e.post("/api/catalog/categories", token, map[string]any{"name": "Accesos y seguridad"}), http.StatusUnprocessableEntity)
		expectStatus(t, e.post("/api/catalog/categories", token, map[string]any{"name": "X", "icon": "bomba"}), http.StatusUnprocessableEntity)
		res := e.post("/api/catalog/categories", token, map[string]any{"name": "Impresión", "icon": "wrench"})
		expectStatus(t, res, http.StatusCreated)
		var c api.ServiceCategory
		res.decode(&c)
		expectStatus(t, e.patch("/api/catalog/categories/"+itoa(c.ID), token, map[string]any{"name": "Impresiones"}), http.StatusOK)
		expectStatus(t, e.do(http.MethodDelete, "/api/catalog/categories/"+itoa(c.ID), token, nil), http.StatusNoContent)
		expectStatus(t, e.do(http.MethodDelete, "/api/catalog/categories/"+itoa(c.ID), token, nil), http.StatusNotFound)
	})
}

func findItem(cats []api.ServiceCategory, id int64) (api.ServiceItem, bool) {
	for _, c := range cats {
		for _, it := range c.Items {
			if it.ID == id {
				return it, true
			}
		}
	}
	return api.ServiceItem{}, false
}

func TestCatalogIsolation(t *testing.T) {
	e := newEnv(t)
	acme := e.signup("Acme", "admin@acme.test", 20)
	beta := e.signup("Beta", "admin@beta.test", 20)
	betaCats := e.catalog(beta.admin.Token, "")
	item := firstItem(t, betaCats, "Alta de nuevo empleado")
	acmeCats := e.catalog(acme.admin.Token, "")
	if acmeCats[0].ID == betaCats[0].ID {
		t.Fatal("cada empresa tiene su propio catálogo")
	}

	ana := e.in(acme).customer("ana")
	expectStatus(t, e.post("/api/catalog/items/"+itoa(item.ID)+"/request", ana.Token,
		map[string]any{"values": map[string]any{"nombre": "x", "puesto": "y", "fecha_ingreso": "2026-01-01"}}), http.StatusNotFound)
	body := map[string]any{"category_id": item.CategoryID, "name": "Colado"}
	expectStatus(t, e.post("/api/catalog/items", acme.admin.Token, body), http.StatusUnprocessableEntity) // categoría ajena
	expectStatus(t, e.patch("/api/catalog/items/"+itoa(item.ID), acme.admin.Token,
		map[string]any{"category_id": acmeCats[0].ID, "name": "Robado"}), http.StatusNotFound)
	expectStatus(t, e.do(http.MethodDelete, "/api/catalog/items/"+itoa(item.ID), acme.admin.Token, nil), http.StatusNotFound)
	expectStatus(t, e.patch("/api/catalog/categories/"+itoa(betaCats[0].ID), acme.admin.Token, map[string]any{"name": "Robada"}), http.StatusNotFound)
	if got := firstItem(t, e.catalog(beta.admin.Token, ""), "Alta de nuevo empleado"); got.Name != "Alta de nuevo empleado" {
		t.Error("el servicio de Beta sigue intacto")
	}
}
