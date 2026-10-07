package api_test

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
)

// tenant es una empresa creada por el registro público, con su administrador.
type tenant struct {
	id    int64
	slug  string
	admin actor
}

func (e *env) signup(company, email string, people int) tenant {
	e.t.Helper()
	res := e.post("/api/signup", "", map[string]any{
		"company": company, "people": people, "name": "Admin " + company, "email": email,
		"password": testPassword, "accept_terms": true,
	})
	expectStatus(e.t, res, http.StatusCreated)
	var out struct {
		Token string   `json:"token"`
		User  api.User `json:"user"`
	}
	res.decode(&out)
	var slug string
	if err := e.pool.QueryRow(context.Background(), `SELECT slug FROM organizations WHERE id = $1`, out.User.OrgID).Scan(&slug); err != nil {
		e.t.Fatal(err)
	}
	return tenant{id: out.User.OrgID, slug: slug, admin: actor{ID: out.User.ID, Name: out.User.Name, Email: email, Token: out.Token}}
}

// in devuelve un entorno que da de alta usuarios en esa empresa.
func (e *env) in(t tenant) *env {
	copy := *e
	copy.org = t.id
	return &copy
}

func (e *env) unlimit(t tenant) {
	e.t.Helper()
	mustExec(e.t, e, `UPDATE organizations SET max_agents = NULL WHERE id = $1`, t.id)
}

func TestSignupCreatesCompanyWithDefaults(t *testing.T) {
	e := newEnv(t)
	acme := e.signup("Ferretería López, S.A.", "rosa@ferreteria.mx", 8)
	if acme.slug != "ferreteria-lopez-s-a" {
		t.Errorf("slug = %q", acme.slug)
	}
	// Un segundo registro con el mismo nombre recibe otro slug.
	if other := e.signup("Ferretería López, S.A.", "otro@ferreteria.mx", 8); other.slug != "ferreteria-lopez-s-a-2" {
		t.Errorf("slug repetido = %q", other.slug)
	}

	res := e.get("/api/org", acme.admin.Token)
	expectStatus(t, res, http.StatusOK)
	var org struct {
		Name      string `json:"name"`
		People    int    `json:"people"`
		MaxAgents *int   `json:"max_agents"`
		Agents    int    `json:"agents"`
		Platform  bool   `json:"platform"`
	}
	res.decode(&org)
	if org.Name != "Ferretería López, S.A." || org.People != 8 || org.MaxAgents == nil || *org.MaxAgents != 1 || org.Agents != 1 || org.Platform {
		t.Errorf("empresa = %+v", org)
	}

	t.Run("trae SLA y categorías para empezar", func(t *testing.T) {
		var sla struct{ Items []api.SLAPolicy }
		e.get("/api/sla", acme.admin.Token).decode(&sla)
		if len(sla.Items) != 4 {
			t.Errorf("SLA = %+v", sla.Items)
		}
		var cats struct{ Items []api.Category }
		e.get("/api/categories", acme.admin.Token).decode(&cats)
		if len(cats.Items) != 3 {
			t.Errorf("categorías = %+v", cats.Items)
		}
	})

	t.Run("validación", func(t *testing.T) {
		res := e.post("/api/signup", "", map[string]any{"company": "!!!", "people": 0, "name": "", "email": "x", "password": "1"})
		expectStatus(t, res, http.StatusUnprocessableEntity)
		for _, f := range []string{"company", "people", "name", "email", "password", "accept_terms"} {
			if _, ok := res.fieldErrors()[f]; !ok {
				t.Errorf("falta error en %s: %s", f, res.Body)
			}
		}
	})

	t.Run("los nombres reservados no se usan como portal", func(t *testing.T) {
		if got := e.signup("Admin", "a@admin.test", 3).slug; got != "admin-soporte" {
			t.Errorf("slug = %q", got)
		}
	})
}

// TestTenantIsolation comprueba endpoint por endpoint que una empresa no ve ni toca lo de otra.
func TestTenantIsolation(t *testing.T) {
	e := newEnv(t)
	a := e.signup("Empresa A", "admin@a.test", 50)
	b := e.signup("Empresa B", "admin@b.test", 50)
	e.unlimit(a)
	e.unlimit(b)
	ea, eb := e.in(a), e.in(b)
	agentA := ea.createUser("Agente A", "agente@a.test", "agent")
	clientA := ea.createUser("Cliente A", "cliente@a.test", "customer")
	agentB := eb.createUser("Agente B", "agente@b.test", "agent")

	// Datos de la empresa A.
	tk := e.newTicket(clientA, map[string]any{"title": "Secreto de A", "category": "Facturación"})
	expectStatus(t, e.patch(ticketPath(tk.ID), agentA.Token, map[string]any{"tags": []string{"privado"}, "assignee_id": agentA.ID}), http.StatusOK)
	res := e.post(ticketPath(tk.ID)+"/comments", agentA.Token, map[string]any{"body": "nota de A", "internal": true})
	expectStatus(t, res, http.StatusCreated)
	upload := e.upload(clientA, tk.ID, "", "a.txt", []byte("archivo de A"))
	expectStatus(t, upload, http.StatusCreated)
	attachmentID := itoa(int64(upload.field("id").(float64)))
	res = e.post("/api/macros", agentA.Token, map[string]any{"title": "Macro A", "body": "texto"})
	macroID := itoa(int64(res.field("id").(float64)))
	res = e.post("/api/articles", agentA.Token, map[string]any{"title": "Borrador de A", "body": "x"})
	draftID := itoa(int64(res.field("id").(float64)))
	res = e.post("/api/articles", agentA.Token, map[string]any{"title": "Guía pública de A", "body": "x", "published": true})
	publicID := itoa(int64(res.field("id").(float64)))
	var catA struct{ Items []api.Category }
	e.get("/api/categories", a.admin.Token).decode(&catA)
	catID := itoa(catA.Items[0].ID)

	for _, intruder := range []actor{agentB, b.admin} {
		t.Run("lo que "+intruder.Name+" no ve", func(t *testing.T) {
			for _, path := range []string{
				ticketPath(tk.ID), ticketPath(tk.ID) + "/comments", ticketPath(tk.ID) + "/events",
				ticketPath(tk.ID) + "/attachments", "/api/attachments/" + attachmentID, "/api/articles/" + draftID,
			} {
				expectStatus(t, e.get(path, intruder.Token), http.StatusNotFound)
			}
			if got := e.list(intruder, ""); len(got.Items) != 0 {
				t.Errorf("lista de tickets = %v", ids(got.Items))
			}
			if got := e.list(intruder, "?q=Secreto"); len(got.Items) != 0 {
				t.Errorf("búsqueda = %v", ids(got.Items))
			}
			for path, forbidden := range map[string]string{
				"/api/users":      "Cliente A",
				"/api/tags":       "privado",
				"/api/macros":     "Macro A",
				"/api/categories": fmt.Sprintf(`"id":%d,`, catA.Items[0].ID),
				"/api/articles":   "de A",
				"/api/reports":    "Agente A",
			} {
				if body := string(e.get(path, intruder.Token).Body); strings.Contains(body, forbidden) {
					t.Errorf("%s muestra datos de A: %s", path, body)
				}
			}
			var stats struct {
				ByStatus map[string]int `json:"by_status"`
			}
			e.get("/api/stats", intruder.Token).decode(&stats)
			if stats.ByStatus["open"] != 0 {
				t.Errorf("stats cuentan tickets de A: %+v", stats.ByStatus)
			}
			csv := string(e.get("/api/reports/tickets.csv", intruder.Token).Body)
			if strings.Contains(csv, "Secreto") {
				t.Error("el CSV exporta tickets de A")
			}
		})

		t.Run("lo que "+intruder.Name+" no puede tocar", func(t *testing.T) {
			expectStatus(t, e.patch(ticketPath(tk.ID), intruder.Token, map[string]any{"status": "closed"}), http.StatusNotFound)
			expectStatus(t, e.post(ticketPath(tk.ID)+"/comments", intruder.Token, map[string]any{"body": "hola"}), http.StatusNotFound)
			expectStatus(t, e.upload(intruder, tk.ID, "", "x.txt", []byte("x")), http.StatusNotFound)
			res := e.post("/api/tickets/bulk", intruder.Token, map[string]any{"ids": []int64{tk.ID}, "status": "closed"})
			if res.field("updated") != float64(0) {
				t.Errorf("bulk modificó un ticket de A: %s", res.Body)
			}
			expectStatus(t, e.patch("/api/macros/"+macroID, intruder.Token, map[string]any{"title": "x", "body": "y"}), http.StatusNotFound)
			expectStatus(t, e.do(http.MethodDelete, "/api/macros/"+macroID, intruder.Token, nil), http.StatusNotFound)
			expectStatus(t, e.patch("/api/articles/"+publicID, intruder.Token, map[string]any{"title": "x"}), http.StatusNotFound)
			expectStatus(t, e.do(http.MethodDelete, "/api/articles/"+publicID, intruder.Token, nil), http.StatusNotFound)
			expectStatus(t, e.do(http.MethodDelete, "/api/attachments/"+attachmentID, intruder.Token, nil), http.StatusNotFound)
		})
	}

	t.Run("el administrador de B no administra gente ni configuración de A", func(t *testing.T) {
		expectStatus(t, e.patch("/api/users/"+itoa(agentA.ID), b.admin.Token, map[string]any{"active": false}), http.StatusNotFound)
		expectStatus(t, e.patch("/api/users/"+itoa(agentA.ID)+"/role", b.admin.Token, map[string]any{"role": "customer"}), http.StatusNotFound)
		expectStatus(t, e.patch("/api/categories/"+catID, b.admin.Token, map[string]any{"name": "Hackeada"}), http.StatusNotFound)
		expectStatus(t, e.do(http.MethodDelete, "/api/categories/"+catID, b.admin.Token, nil), http.StatusNotFound)
		body := map[string]any{"items": []map[string]any{{"priority": "urgent", "first_response_minutes": 1, "resolution_minutes": 2}}}
		expectStatus(t, e.do(http.MethodPut, "/api/sla", b.admin.Token, body), http.StatusOK)
		var sla struct{ Items []api.SLAPolicy }
		e.get("/api/sla", a.admin.Token).decode(&sla)
		if sla.Items[0].FirstResponseMinutes != 60 {
			t.Errorf("el SLA de A cambió: %+v", sla.Items[0])
		}
	})

	t.Run("no se asigna a un agente ni se abre en nombre de alguien de otra empresa", func(t *testing.T) {
		mine := e.newTicket(b.admin, map[string]any{"title": "De B"})
		expectFieldError(t, e.patch(ticketPath(mine.ID), b.admin.Token, map[string]any{"assignee_id": agentA.ID}), "assignee_id")
		expectFieldError(t, e.post("/api/tickets", b.admin.Token, map[string]any{"title": "x", "requester_id": clientA.ID}), "requester_id")
		expectFieldError(t, e.post("/api/tickets", b.admin.Token, map[string]any{"title": "x", "category": "Inventada"}), "category")
	})

	t.Run("las menciones y avisos no cruzan empresas", func(t *testing.T) {
		mine := e.newTicket(b.admin, map[string]any{"title": "Mención"})
		e.post(ticketPath(mine.ID)+"/comments", b.admin.Token, map[string]any{"body": "@Agente A mira", "internal": true})
		for _, n := range e.notifications(agentA).Items {
			if n.TicketID != nil && *n.TicketID == mine.ID {
				t.Errorf("A recibió un aviso de B: %+v", n)
			}
		}
	})

	t.Run("la ayuda publicada de cada empresa es pública y separada", func(t *testing.T) {
		var list struct{ Items []api.Article }
		e.get("/api/articles?org="+a.slug, "").decode(&list)
		if len(list.Items) != 1 || list.Items[0].Title != "Guía pública de A" {
			t.Errorf("ayuda de A = %+v", list.Items)
		}
		e.get("/api/articles?org="+b.slug, "").decode(&list)
		if len(list.Items) != 0 {
			t.Errorf("ayuda de B = %+v", list.Items)
		}
		expectStatus(t, e.get("/api/articles/"+publicID, ""), http.StatusOK)
		expectStatus(t, e.get("/api/articles?org=no-existe", ""), http.StatusNotFound)
	})
}

func TestPortalRegistrationAndMultiCompanyLogin(t *testing.T) {
	e := newEnv(t)
	a := e.signup("Panadería Sol", "sol@pan.test", 5)
	b := e.signup("Taller Luna", "luna@taller.test", 5)

	res := e.get("/api/portal/"+a.slug, "")
	expectStatus(t, res, http.StatusOK)
	if res.field("name") != "Panadería Sol" {
		t.Errorf("portal = %s", res.Body)
	}
	expectStatus(t, e.get("/api/portal/no-existe", ""), http.StatusNotFound)

	// La misma persona es clienta de las dos empresas, con la misma contraseña.
	for _, org := range []tenant{a, b} {
		res := e.post("/api/auth/register", "", map[string]any{"name": "Ana", "email": "ana@correo.test", "password": testPassword, "org": org.slug})
		expectStatus(t, res, http.StatusCreated)
		if role := res.field("user").(map[string]any)["role"]; role != "customer" {
			t.Errorf("rol = %v", role)
		}
	}
	expectStatus(t, e.post("/api/auth/register", "", map[string]any{"name": "Ana", "email": "ANA@correo.test", "password": testPassword, "org": a.slug}), http.StatusConflict)

	t.Run("sin elegir empresa, el login pide escoger", func(t *testing.T) {
		res := e.post("/api/auth/login", "", map[string]any{"email": "ana@correo.test", "password": testPassword})
		expectStatus(t, res, http.StatusConflict)
		orgs := res.field("organizations").([]any)
		if len(orgs) != 2 {
			t.Fatalf("empresas = %v", orgs)
		}
	})
	t.Run("eligiendo empresa entra a esa", func(t *testing.T) {
		res := e.post("/api/auth/login", "", map[string]any{"email": "ana@correo.test", "password": testPassword, "org": b.slug})
		expectStatus(t, res, http.StatusOK)
		if org := int64(res.field("user").(map[string]any)["org_id"].(float64)); org != b.id {
			t.Errorf("entró a la empresa %d", org)
		}
	})
	t.Run("la contraseña incorrecta no revela en qué empresas hay cuenta", func(t *testing.T) {
		res := e.post("/api/auth/login", "", map[string]any{"email": "ana@correo.test", "password": "incorrecta"})
		expectStatus(t, res, http.StatusUnauthorized)
		if strings.Contains(string(res.Body), a.slug) {
			t.Error("revela empresas")
		}
	})
}

func TestAgentLimitAndUpgradeRequest(t *testing.T) {
	e := newEnv(t)
	owner := e.createUser("Dueño", "madgrismad@gmail.com", "admin")
	free := e.signup("Changarro", "dueno@changarro.test", 5)
	ef := e.in(free)
	customer := ef.createUser("Cliente", "cliente@changarro.test", "customer")

	t.Run("el plan gratis incluye 1 agente", func(t *testing.T) {
		res := e.post("/api/users", free.admin.Token, map[string]any{"name": "Agente", "email": "ag@changarro.test", "role": "agent", "password": "inicial123"})
		expectFieldError(t, res, "role")
		if !strings.Contains(res.fieldErrors()["role"].(string), "1 agente") {
			t.Errorf("mensaje = %s", res.Body)
		}
		expectFieldError(t, e.patch("/api/users/"+itoa(customer.ID), free.admin.Token, map[string]any{"role": "agent"}), "role")
		expectFieldError(t, e.patch("/api/users/"+itoa(customer.ID)+"/role", free.admin.Token, map[string]any{"role": "admin"}), "role")
		// Los clientes no cuentan.
		expectStatus(t, e.post("/api/users", free.admin.Token, map[string]any{"name": "Otro", "email": "c2@changarro.test", "role": "customer", "password": "inicial123"}), http.StatusCreated)
	})

	t.Run("pedir ampliar el plan avisa a la plataforma", func(t *testing.T) {
		expectStatus(t, e.post("/api/org/upgrade", free.admin.Token, map[string]any{"agents": 3, "people": 20, "message": "Crecimos"}), http.StatusCreated)
		var leads struct{ Items []api.Lead }
		e.get("/api/leads", owner.Token).decode(&leads)
		if len(leads.Items) != 1 || leads.Items[0].Company != "Changarro" || leads.Items[0].Agents != 3 ||
			leads.Items[0].OrgID == nil || *leads.Items[0].OrgID != free.id {
			t.Errorf("solicitud = %+v", leads.Items)
		}
		if n := e.notifications(owner).Items; len(n) == 0 || !strings.Contains(n[0].Summary, "ampliar su plan") {
			t.Errorf("aviso = %+v", n)
		}
	})

	t.Run("la plataforma amplía el plan y ya caben más agentes", func(t *testing.T) {
		expectStatus(t, e.patch("/api/platform/orgs/"+itoa(free.id), free.admin.Token, map[string]any{"max_agents": 3}), http.StatusForbidden)
		expectStatus(t, e.patch("/api/platform/orgs/"+itoa(free.id), owner.Token, map[string]any{"max_agents": 3, "people": 20}), http.StatusOK)
		for i := range 2 {
			res := e.post("/api/users", free.admin.Token, map[string]any{"name": "Agente", "email": fmt.Sprintf("ag%d@changarro.test", i), "role": "agent", "password": "inicial123"})
			expectStatus(t, res, http.StatusCreated)
		}
		expectFieldError(t, e.post("/api/users", free.admin.Token, map[string]any{"name": "Uno más", "email": "ag9@changarro.test", "role": "agent", "password": "inicial123"}), "role")
	})

	t.Run("un agente desactivado libera su lugar", func(t *testing.T) {
		var users struct{ Items []api.User }
		e.get("/api/users?role=agent", free.admin.Token).decode(&users)
		expectStatus(t, e.patch("/api/users/"+itoa(users.Items[0].ID), free.admin.Token, map[string]any{"active": false}), http.StatusOK)
		expectStatus(t, e.post("/api/users", free.admin.Token, map[string]any{"name": "Nueva", "email": "nueva@changarro.test", "role": "agent", "password": "inicial123"}), http.StatusCreated)
		// Y reactivarlo ya no cabe.
		expectFieldError(t, e.patch("/api/users/"+itoa(users.Items[0].ID), free.admin.Token, map[string]any{"active": true}), "active")
	})
}

func TestPlatformPanelAndSuspension(t *testing.T) {
	e := newEnv(t)
	owner := e.createUser("Dueño", "madgrismad@gmail.com", "admin")
	platformAdmin := e.admin("otro-admin")
	acme := e.signup("Acme", "admin@acme.test", 30)

	expectStatus(t, e.get("/api/platform/orgs", platformAdmin.Token), http.StatusForbidden)
	expectStatus(t, e.get("/api/platform/orgs", acme.admin.Token), http.StatusForbidden)
	var list struct {
		Items []struct {
			ID     int64 `json:"id"`
			Name   string
			Agents int
		}
	}
	e.get("/api/platform/orgs", owner.Token).decode(&list)
	if len(list.Items) != 2 || list.Items[0].Name != "Acme" || list.Items[0].Agents != 1 {
		t.Errorf("empresas = %+v", list.Items)
	}

	expectFieldError(t, e.patch("/api/platform/orgs/1", owner.Token, map[string]any{"suspended": true}), "suspended")
	expectStatus(t, e.patch("/api/platform/orgs/"+itoa(acme.id), owner.Token, map[string]any{"suspended": true}), http.StatusOK)

	expectStatus(t, e.get("/api/me", acme.admin.Token), http.StatusUnauthorized)
	expectStatus(t, e.post("/api/auth/login", "", map[string]any{"email": "admin@acme.test", "password": testPassword}), http.StatusForbidden)
	expectStatus(t, e.get("/api/portal/"+acme.slug, ""), http.StatusNotFound)

	expectStatus(t, e.patch("/api/platform/orgs/"+itoa(acme.id), owner.Token, map[string]any{"suspended": false}), http.StatusOK)
	expectStatus(t, e.post("/api/auth/login", "", map[string]any{"email": "admin@acme.test", "password": testPassword}), http.StatusOK)
}

func TestOneSessionPerAgent(t *testing.T) {
	e := newEnv(t)
	luis := e.agent("luis")
	ana := e.customer("ana")

	time.Sleep(1100 * time.Millisecond) // los tokens de prueba son de un segundo anterior
	res := e.post("/api/auth/login", "", map[string]any{"email": luis.Email, "password": testPassword})
	expectStatus(t, res, http.StatusOK)
	newToken := res.field("token").(string)

	old := e.get("/api/me", luis.Token)
	expectStatus(t, old, http.StatusUnauthorized)
	if !strings.Contains(old.errorText(), "otro dispositivo") {
		t.Errorf("mensaje = %q", old.errorText())
	}
	expectStatus(t, e.get("/api/me", newToken), http.StatusOK)

	t.Run("los clientes pueden tener varias sesiones", func(t *testing.T) {
		expectStatus(t, e.post("/api/auth/login", "", map[string]any{"email": ana.Email, "password": testPassword}), http.StatusOK)
		expectStatus(t, e.get("/api/me", ana.Token), http.StatusOK)
	})
}

func TestStreamDoesNotCrossCompanies(t *testing.T) {
	e := newEnv(t)
	a := e.signup("Empresa A", "admin@a.test", 5)
	b := e.signup("Empresa B", "admin@b.test", 5)
	srv := httptest.NewServer(e.handler)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/stream", nil)
	req.Header.Set("Authorization", "Bearer "+b.admin.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	reader := bufio.NewReader(res.Body)

	// Primero cambia un ticket de A y luego uno de B: el primer aviso que recibe B debe ser el suyo.
	tkA := e.newTicket(a.admin, map[string]any{"title": "de A"})
	tkB := e.newTicket(b.admin, map[string]any{"title": "de B"})
	got := make(chan string, 1)
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				got <- "error"
				return
			}
			if strings.HasPrefix(line, "data: ") && strings.Contains(line, `"type":"ticket"`) {
				got <- line
				return
			}
		}
	}()
	select {
	case line := <-got:
		if strings.Contains(line, `"ticket_id":`+itoa(tkA.ID)+`}`) {
			t.Fatalf("B recibió un aviso del ticket de A: %s", line)
		}
		if !strings.Contains(line, `"ticket_id":`+itoa(tkB.ID)) {
			t.Errorf("aviso inesperado: %s", line)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no llegó ningún aviso")
	}
}
