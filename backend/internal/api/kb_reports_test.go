package api_test

import (
	"encoding/csv"
	"net/http"
	"strings"
	"testing"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
)

func TestKnowledgeBase(t *testing.T) {
	e := newEnv(t)
	luis := e.agent("luis")
	ana := e.customer("ana")

	res := e.post("/api/articles", luis.Token, map[string]any{
		"title": "Cómo configurar la impresora", "body": "Instala el controlador de las impresoras HP.", "published": true,
	})
	expectStatus(t, res, http.StatusCreated)
	var pub api.Article
	res.decode(&pub)
	res = e.post("/api/articles", luis.Token, map[string]any{"title": "Borrador interno", "body": "Pendiente"})
	expectStatus(t, res, http.StatusCreated)
	var draft api.Article
	res.decode(&draft)

	type list struct {
		Items []api.Article `json:"items"`
	}
	t.Run("sin sesión solo se ven los publicados", func(t *testing.T) {
		var l list
		e.get("/api/articles", "").decode(&l)
		if len(l.Items) != 1 || l.Items[0].ID != pub.ID {
			t.Errorf("artículos públicos = %+v", l.Items)
		}
		expectStatus(t, e.get("/api/articles/"+itoa(draft.ID), ""), http.StatusNotFound)
		expectStatus(t, e.get("/api/articles/"+itoa(draft.ID), ana.Token), http.StatusNotFound)
		expectStatus(t, e.get("/api/articles/"+itoa(draft.ID), luis.Token), http.StatusOK)
		expectStatus(t, e.get("/api/articles/"+itoa(pub.ID), ""), http.StatusOK)
	})
	t.Run("un token inválido se trata como visitante", func(t *testing.T) {
		expectStatus(t, e.get("/api/articles", "basura"), http.StatusOK)
	})
	t.Run("búsqueda en español", func(t *testing.T) {
		var l list
		e.get("/api/articles?q=impresoras", ana.Token).decode(&l)
		if len(l.Items) != 1 {
			t.Errorf("búsqueda = %+v", l.Items)
		}
		e.get("/api/articles?q=borrador", luis.Token).decode(&l)
		if len(l.Items) != 1 || l.Items[0].ID != draft.ID {
			t.Errorf("los agentes buscan también borradores: %+v", l.Items)
		}
	})
	t.Run("solo agentes editan", func(t *testing.T) {
		expectStatus(t, e.post("/api/articles", ana.Token, map[string]any{"title": "x"}), http.StatusForbidden)
		expectFieldError(t, e.post("/api/articles", luis.Token, map[string]any{"title": " "}), "title")
		res := e.patch("/api/articles/"+itoa(draft.ID), luis.Token, map[string]any{"title": "Ya listo", "body": "x", "published": true})
		expectStatus(t, res, http.StatusOK)
		expectStatus(t, e.get("/api/articles/"+itoa(draft.ID), ""), http.StatusOK)
		expectStatus(t, e.do(http.MethodDelete, "/api/articles/"+itoa(draft.ID), luis.Token, nil), http.StatusNoContent)
		expectStatus(t, e.get("/api/articles/"+itoa(draft.ID), luis.Token), http.StatusNotFound)
	})
}

func TestSavedViews(t *testing.T) {
	e := newEnv(t)
	luis := e.agent("luis")
	marta := e.agent("marta")

	res := e.post("/api/views", luis.Token, map[string]any{"name": "Urgentes míos", "query": "?priority=urgent&assignee=me"})
	expectStatus(t, res, http.StatusCreated)
	var v api.SavedView
	res.decode(&v)
	if v.Query != "?assignee=me&priority=urgent" {
		t.Errorf("query normalizada = %q", v.Query)
	}
	expectFieldError(t, e.post("/api/views", luis.Token, map[string]any{"name": "x", "query": "?hack=1"}), "query")
	expectFieldError(t, e.post("/api/views", luis.Token, map[string]any{"name": "x", "query": ""}), "query")
	expectFieldError(t, e.post("/api/views", luis.Token, map[string]any{"name": "", "query": "?q=a"}), "name")

	var l struct {
		Items []api.SavedView `json:"items"`
	}
	e.get("/api/views", marta.Token).decode(&l)
	if len(l.Items) != 0 {
		t.Error("las vistas son personales")
	}
	expectStatus(t, e.do(http.MethodDelete, "/api/views/"+itoa(v.ID), marta.Token, nil), http.StatusNotFound)
	expectStatus(t, e.do(http.MethodDelete, "/api/views/"+itoa(v.ID), luis.Token, nil), http.StatusNoContent)
}

func TestReports(t *testing.T) {
	e := newEnv(t)
	luis := e.agent("luis")
	ana := e.customer("ana")

	a := e.newTicket(ana, map[string]any{"title": "=HYPERLINK(\"http://malo\")"})
	e.newTicket(ana, map[string]any{"title": "Otro"})
	e.post(ticketPath(a.ID)+"/comments", luis.Token, map[string]any{"body": "Hola"})
	e.patch(ticketPath(a.ID), luis.Token, map[string]any{"assignee_id": luis.ID, "status": "resolved"})
	e.post(ticketPath(a.ID)+"/satisfaction", ana.Token, map[string]any{"rating": "good"})

	res := e.get("/api/reports?tz=America/Mexico_City", luis.Token)
	expectStatus(t, res, http.StatusOK)
	var rep struct {
		Created, Resolved   int
		Good                int     `json:"satisfaction_good"`
		FirstResponseSLAMet float64 `json:"first_response_sla_met"`
		Daily               []struct{ Date string }
		Agents              []struct {
			Name     string
			Resolved int
		}
	}
	res.decode(&rep)
	if rep.Created != 2 || rep.Resolved != 1 || rep.Good != 1 || rep.FirstResponseSLAMet != 1 {
		t.Errorf("reporte = %+v", rep)
	}
	if len(rep.Daily) != 30 {
		t.Errorf("días = %d", len(rep.Daily))
	}
	if len(rep.Agents) != 1 || rep.Agents[0].Resolved != 1 {
		t.Errorf("agentes = %+v", rep.Agents)
	}

	expectStatus(t, e.get("/api/reports?tz=Marte/Olympus", luis.Token), http.StatusBadRequest)
	expectStatus(t, e.get("/api/reports?from=2026-13-01", luis.Token), http.StatusBadRequest)
	expectStatus(t, e.get("/api/reports?from=2026-02-01&to=2026-01-01", luis.Token), http.StatusBadRequest)
	expectStatus(t, e.get("/api/reports", ana.Token), http.StatusForbidden)

	t.Run("CSV", func(t *testing.T) {
		res := e.get("/api/reports/tickets.csv", luis.Token)
		expectStatus(t, res, http.StatusOK)
		if !strings.HasPrefix(res.Header.Get("Content-Disposition"), "attachment") {
			t.Error("debe descargarse")
		}
		records, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(res.Body), "\ufeff"))).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 3 || records[0][1] != "Título" {
			t.Fatalf("filas = %v", records)
		}
		if got := records[1][1]; !strings.HasPrefix(got, "'=") {
			t.Errorf("las fórmulas deben neutralizarse: %q", got)
		}
		if records[1][11] != "Buena" || records[1][2] != "resuelto" {
			t.Errorf("fila = %v", records[1])
		}
	})
}
