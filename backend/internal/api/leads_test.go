package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
)

func TestLeads(t *testing.T) {
	e := newEnv(t)
	// Las solicitudes las gestionan los administradores permanentes de la plataforma.
	boss := e.createUser("Dueño", "madgrismad@gmail.com", "admin")
	otherAdmin := e.admin("jefa")
	luis := e.agent("luis")

	lead := map[string]any{
		"name": "Rosa Méndez", "company": "Ferretería López", "email": "rosa@ferreteria.mx",
		"phone": "+52 33 1234 5678", "agents": 6, "people": 120, "message": "Queremos dejar el correo.",
	}
	expectStatus(t, e.post("/api/leads", "", lead), http.StatusCreated)

	t.Run("avisa a los administradores", func(t *testing.T) {
		n := e.notifications(boss)
		if len(n.Items) != 1 || n.Items[0].Kind != "lead" || !strings.Contains(n.Items[0].Summary, "Ferretería López") {
			t.Errorf("notificaciones = %+v", n.Items)
		}
		if len(e.notifications(luis).Items) != 0 {
			t.Error("los agentes no reciben solicitudes de demo")
		}
		var body string
		e.pool.QueryRow(context.Background(), `SELECT body FROM email_outbox`).Scan(&body)
		if !strings.Contains(body, "Agentes: 6") || !strings.Contains(body, "Personas en la empresa: 120") {
			t.Errorf("el correo debe incluir lo elegido en la calculadora:\n%s", body)
		}
		// Llega a todos los administradores de la plataforma.
		if mails := e.outbox(); len(mails) != 2 || !strings.Contains(strings.Join(mails, "\n"), boss.Email) {
			t.Errorf("correos = %v", mails)
		}
	})

	t.Run("solo los administradores las ven y las marcan", func(t *testing.T) {
		expectStatus(t, e.get("/api/leads", luis.Token), http.StatusForbidden)
		expectStatus(t, e.get("/api/leads", otherAdmin.Token), http.StatusForbidden)
		var out struct {
			Items []api.Lead `json:"items"`
		}
		e.get("/api/leads", boss.Token).decode(&out)
		if len(out.Items) != 1 || out.Items[0].Company != "Ferretería López" || out.Items[0].Handled ||
			out.Items[0].Agents != 6 || out.Items[0].People != 120 {
			t.Fatalf("leads = %+v", out.Items)
		}
		res := e.patch("/api/leads/"+itoa(out.Items[0].ID), boss.Token, map[string]any{"handled": true})
		expectStatus(t, res, http.StatusOK)
		if res.field("handled") != true {
			t.Errorf("handled = %v", res.field("handled"))
		}
		expectStatus(t, e.patch("/api/leads/999999", boss.Token, map[string]any{"handled": true}), http.StatusNotFound)
	})

	t.Run("validación", func(t *testing.T) {
		res := e.post("/api/leads", "", map[string]any{"name": "", "company": "", "email": "x", "team_size": "mil", "plan": "oro", "agents": -1, "people": -5})
		expectStatus(t, res, http.StatusUnprocessableEntity)
		for _, f := range []string{"name", "company", "email", "team_size", "plan", "agents", "people"} {
			if _, ok := res.fieldErrors()[f]; !ok {
				t.Errorf("falta error en %s: %s", f, res.Body)
			}
		}
	})

	t.Run("el campo trampa descarta a los bots sin avisarles", func(t *testing.T) {
		bot := map[string]any{"name": "Bot", "company": "Spam", "email": "bot@spam.test", "website": "http://spam"}
		expectStatus(t, e.post("/api/leads", "", bot), http.StatusCreated)
		var out struct {
			Items []api.Lead `json:"items"`
		}
		e.get("/api/leads", boss.Token).decode(&out)
		if len(out.Items) != 1 {
			t.Errorf("el bot no debía guardarse: %+v", out.Items)
		}
	})
}
