package api_test

import (
	"net/http"
	"testing"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
)

func (e *env) newAsset(token string, body map[string]any) api.Asset {
	e.t.Helper()
	if _, ok := body["name"]; !ok {
		body["name"] = "Laptop"
	}
	res := e.post("/api/assets", token, body)
	if res.Status != http.StatusCreated {
		e.t.Fatalf("crear equipo: %d %s", res.Status, res.Body)
	}
	var a api.Asset
	res.decode(&a)
	return a
}

type assetDetail struct {
	Asset  api.Asset         `json:"asset"`
	Events []api.AssetEvent  `json:"events"`
	Ticket []api.AssetTicket `json:"tickets"`
}

func (e *env) assetDetail(token string, id int64) assetDetail {
	e.t.Helper()
	res := e.get("/api/assets/"+itoa(id), token)
	expectStatus(e.t, res, http.StatusOK)
	var d assetDetail
	res.decode(&d)
	return d
}

func TestAssets(t *testing.T) {
	e := newEnv(t)
	acme := e.signup("Acme", "admin@acme.test", 20)
	ae := e.in(acme)
	agent := ae.agent("agente")
	ana := ae.customer("ana")
	luis := ae.customer("luis")
	e.unlimit(acme)

	t.Run("solo el equipo de soporte ve el inventario", func(t *testing.T) {
		expectStatus(t, e.get("/api/assets", ana.Token), http.StatusForbidden)
		expectStatus(t, e.post("/api/assets", ana.Token, map[string]any{"tag": "X", "name": "X"}), http.StatusForbidden)
		expectStatus(t, e.get("/api/assets/summary", ana.Token), http.StatusForbidden)
	})

	t.Run("validación y etiqueta única", func(t *testing.T) {
		res := e.post("/api/assets", agent.Token, map[string]any{
			"category": "nave", "state": "roto", "purchase_date": "ayer", "purchase_cost": -5, "assigned_to": 99999})
		expectStatus(t, res, http.StatusUnprocessableEntity)
		for _, f := range []string{"tag", "name", "category", "state", "purchase_date", "purchase_cost"} {
			if _, ok := res.fieldErrors()[f]; !ok {
				t.Errorf("falta error en %s: %s", f, res.Body)
			}
		}
		e.newAsset(agent.Token, map[string]any{"tag": "LAP-001"})
		res = e.post("/api/assets", agent.Token, map[string]any{"tag": "lap-001", "name": "Otra"})
		expectStatus(t, res, http.StatusUnprocessableEntity)
		expectFieldError(t, res, "tag")
		res = e.post("/api/assets", agent.Token, map[string]any{"tag": "R", "name": "R", "state": "retired", "assigned_to": ana.ID})
		expectFieldError(t, res, "assigned_to")
	})

	laptop := e.newAsset(agent.Token, map[string]any{
		"tag": "LAP-002", "name": "Dell Latitude 5350", "model": "Latitude 5350", "serial": "ABC123",
		"purchase_date": "2024-01-10", "purchase_cost": 18500.5, "warranty_until": "2099-01-01", "location": "Oficina",
	})
	if laptop.State != "in_stock" || laptop.AssignedTo != nil || laptop.PurchaseCost == nil || *laptop.PurchaseCost != 18500.5 ||
		laptop.WarrantyUntil == nil || *laptop.WarrantyUntil != "2099-01-01" {
		t.Fatalf("equipo = %+v", laptop)
	}

	t.Run("asignarlo lo deja en uso y queda en la actividad", func(t *testing.T) {
		res := e.patch("/api/assets/"+itoa(laptop.ID), agent.Token, map[string]any{
			"tag": laptop.Tag, "name": laptop.Name, "category": "computer", "assigned_to": ana.ID,
			"location": "Ventas", "warranty_until": "2099-01-01", "purchase_date": "2024-01-10", "purchase_cost": 18500.5,
			"model": "Latitude 5350", "serial": "ABC123"})
		expectStatus(t, res, http.StatusOK)
		d := e.assetDetail(agent.Token, laptop.ID)
		if d.Asset.State != "in_use" || d.Asset.AssignedTo == nil || d.Asset.AssignedTo.ID != ana.ID {
			t.Fatalf("equipo = %+v", d.Asset)
		}
		kinds := map[string]string{}
		for _, ev := range d.Events {
			kinds[ev.Kind] = ev.NewValue
		}
		if kinds["assigned"] != "ana" || kinds["state"] != "in_use" || kinds["location"] != "Ventas" || kinds["created"] != "" {
			t.Errorf("actividad = %+v", d.Events)
		}
		if _, noisy := kinds["updated"]; noisy {
			t.Errorf("no se registran cambios que no ocurrieron: %+v", d.Events)
		}
	})

	t.Run("mis equipos y tickets sobre un equipo", func(t *testing.T) {
		var mine struct{ Items []api.Asset }
		e.get("/api/assets/mine", ana.Token).decode(&mine)
		if len(mine.Items) != 1 || mine.Items[0].ID != laptop.ID {
			t.Fatalf("mis equipos = %+v", mine.Items)
		}
		e.get("/api/assets/mine", luis.Token).decode(&mine)
		if len(mine.Items) != 0 {
			t.Errorf("luis no tiene equipos: %+v", mine.Items)
		}

		// Un cliente solo puede reportar un problema con un equipo suyo.
		res := e.post("/api/tickets", luis.Token, map[string]any{"title": "Falla", "asset_id": laptop.ID})
		expectStatus(t, res, http.StatusUnprocessableEntity)
		expectFieldError(t, res, "asset_id")
		tk := ae.newTicket(ana, map[string]any{"title": "No enciende", "asset_id": laptop.ID})

		d := e.assetDetail(agent.Token, laptop.ID)
		if len(d.Ticket) != 1 || d.Ticket[0].ID != tk.ID || d.Asset.OpenTickets != 1 {
			t.Errorf("tickets del equipo = %+v (abiertos %d)", d.Ticket, d.Asset.OpenTickets)
		}
		got := ae.list(agent, "?asset="+itoa(laptop.ID))
		if len(got.Items) != 1 || got.Items[0].ID != tk.ID {
			t.Errorf("filtro por equipo = %v", ids(got.Items))
		}

		// El soporte vincula y desvincula a mano.
		other := ae.newTicket(luis, map[string]any{"title": "Otro problema"})
		expectStatus(t, e.post("/api/tickets/"+itoa(other.ID)+"/assets", agent.Token, map[string]any{"asset_id": laptop.ID}), http.StatusOK)
		expectStatus(t, e.post("/api/tickets/"+itoa(other.ID)+"/assets", ana.Token, map[string]any{"asset_id": laptop.ID}), http.StatusForbidden)
		var linked struct{ Items []api.Asset }
		e.get("/api/tickets/"+itoa(other.ID)+"/assets", agent.Token).decode(&linked)
		if len(linked.Items) != 1 {
			t.Errorf("vinculados = %+v", linked.Items)
		}
		expectStatus(t, e.do(http.MethodDelete, "/api/tickets/"+itoa(other.ID)+"/assets/"+itoa(laptop.ID), agent.Token, nil), http.StatusNoContent)
		expectStatus(t, e.do(http.MethodDelete, "/api/tickets/"+itoa(other.ID)+"/assets/"+itoa(laptop.ID), agent.Token, nil), http.StatusNotFound)

		// Resolver el ticket baja el contador de abiertos.
		expectStatus(t, e.patch("/api/tickets/"+itoa(tk.ID), agent.Token, map[string]any{"status": "resolved"}), http.StatusOK)
		if d := e.assetDetail(agent.Token, laptop.ID); d.Asset.OpenTickets != 0 {
			t.Errorf("abiertos = %d", d.Asset.OpenTickets)
		}
	})

	t.Run("filtros y búsqueda", func(t *testing.T) {
		e.newAsset(agent.Token, map[string]any{"tag": "PH-001", "name": "iPhone 15", "category": "phone", "warranty_until": "2020-01-01"})
		e.newAsset(agent.Token, map[string]any{"tag": "MON-001", "name": "Monitor 24", "category": "monitor",
			"state": "in_repair", "warranty_until": "2099-12-31"})
		count := func(query string) int {
			t.Helper()
			res := e.get("/api/assets?"+query, agent.Token)
			expectStatus(t, res, http.StatusOK)
			var out struct{ Items []api.Asset }
			res.decode(&out)
			return len(out.Items)
		}
		for query, want := range map[string]int{
			"": 4, "category=phone": 1, "state=in_repair": 1, "state=in_use": 1, "assigned=none": 3,
			"assigned=me": 0, "assigned=" + itoa(ana.ID): 1, "q=iphone": 1, "q=ana": 1, "q=ABC123": 1,
			"q=%25": 0, "warranty=expired": 1, "limit=2": 2,
		} {
			if got := count(query); got != want {
				t.Errorf("?%s: %d, se esperaban %d", query, got, want)
			}
		}
		expectStatus(t, e.get("/api/assets?state=x", agent.Token), http.StatusBadRequest)
		expectStatus(t, e.get("/api/assets?warranty=x", agent.Token), http.StatusBadRequest)

		var page struct {
			Items []api.Asset
			Next  *int64 `json:"next_cursor"`
		}
		e.get("/api/assets?limit=3", agent.Token).decode(&page)
		if len(page.Items) != 3 || page.Next == nil {
			t.Fatalf("página = %+v", page)
		}
		e.get("/api/assets?limit=3&before="+itoa(*page.Next), agent.Token).decode(&page)
		if len(page.Items) != 1 || page.Next != nil {
			t.Errorf("segunda página = %+v", page)
		}
	})

	t.Run("resumen", func(t *testing.T) {
		var sum struct {
			Total           int            `json:"total"`
			TotalValue      float64        `json:"total_value"`
			ByState         map[string]int `json:"by_state"`
			ByCategory      map[string]int `json:"by_category"`
			WarrantyExpired int            `json:"warranty_expired"`
		}
		e.get("/api/assets/summary", agent.Token).decode(&sum)
		if sum.Total != 4 || sum.TotalValue != 18500.5 || sum.ByState["in_repair"] != 1 || sum.ByCategory["phone"] != 1 || sum.WarrantyExpired != 1 {
			t.Errorf("resumen = %+v", sum)
		}
	})

	t.Run("dar de baja y borrar", func(t *testing.T) {
		expectStatus(t, e.do(http.MethodDelete, "/api/assets/"+itoa(laptop.ID), agent.Token, nil), http.StatusForbidden)
		expectStatus(t, e.do(http.MethodDelete, "/api/assets/"+itoa(laptop.ID), acme.admin.Token, nil), http.StatusNoContent)
		expectStatus(t, e.get("/api/assets/"+itoa(laptop.ID), agent.Token), http.StatusNotFound)
	})
}

func TestAssetsIsolation(t *testing.T) {
	e := newEnv(t)
	acme := e.signup("Acme", "admin@acme.test", 20)
	beta := e.signup("Beta", "admin@beta.test", 20)
	asset := e.newAsset(beta.admin.Token, map[string]any{"tag": "LAP-001"})
	// La misma etiqueta puede existir en otra empresa.
	mine := e.newAsset(acme.admin.Token, map[string]any{"tag": "LAP-001"})

	expectStatus(t, e.get("/api/assets/"+itoa(asset.ID), acme.admin.Token), http.StatusNotFound)
	expectStatus(t, e.patch("/api/assets/"+itoa(asset.ID), acme.admin.Token,
		map[string]any{"tag": "X", "name": "Robado"}), http.StatusNotFound)
	expectStatus(t, e.do(http.MethodDelete, "/api/assets/"+itoa(asset.ID), acme.admin.Token, nil), http.StatusNotFound)

	var list struct{ Items []api.Asset }
	e.get("/api/assets", acme.admin.Token).decode(&list)
	if len(list.Items) != 1 || list.Items[0].ID != mine.ID {
		t.Errorf("lista = %+v", list.Items)
	}
	var sum struct{ Total int }
	e.get("/api/assets/summary", acme.admin.Token).decode(&sum)
	if sum.Total != 1 {
		t.Errorf("resumen = %+v", sum)
	}

	// No se puede vincular un equipo de otra empresa, ni asignar a un usuario ajeno.
	ana := e.in(acme).customer("ana")
	tk := e.in(acme).newTicket(ana, map[string]any{"title": "Falla"})
	res := e.post("/api/tickets/"+itoa(tk.ID)+"/assets", acme.admin.Token, map[string]any{"asset_id": asset.ID})
	expectStatus(t, res, http.StatusUnprocessableEntity)
	res = e.post("/api/tickets", acme.admin.Token, map[string]any{"title": "x", "asset_id": asset.ID})
	expectFieldError(t, res, "asset_id")
	outsider := e.in(beta).customer("ajeno")
	res = e.post("/api/assets", acme.admin.Token, map[string]any{"tag": "Z", "name": "Z", "assigned_to": outsider.ID})
	expectFieldError(t, res, "assigned_to")
	if d := e.assetDetail(beta.admin.Token, asset.ID); len(d.Ticket) != 0 || len(d.Events) != 1 {
		t.Errorf("el equipo de Beta no cambió: %+v", d)
	}
}
