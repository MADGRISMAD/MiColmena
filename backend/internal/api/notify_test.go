package api_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
)

type notifList struct {
	Items  []api.Notification `json:"items"`
	Unread int                `json:"unread"`
}

func (e *env) notifications(a actor) notifList {
	e.t.Helper()
	res := e.get("/api/notifications", a.Token)
	expectStatus(e.t, res, http.StatusOK)
	var out notifList
	res.decode(&out)
	return out
}

func notifKinds(l notifList) []string {
	out := []string{}
	for _, n := range l.Items {
		out = append(out, n.Kind)
	}
	return out
}

// outbox devuelve los destinatarios y asuntos de los correos en cola.
func (e *env) outbox() []string {
	e.t.Helper()
	rows, err := e.pool.Query(context.Background(), `SELECT to_email || ' | ' || subject FROM email_outbox ORDER BY id`)
	if err != nil {
		e.t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

func TestNotificationsFlow(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.createUser("Luis Ramírez", "luis@equipo.test", "agent")
	marta := e.createUser("Marta", "marta@equipo.test", "agent")
	mustExec(t, e, `UPDATE users SET email_notifications = false WHERE id = $1`, marta.ID)

	tk := e.newTicket(ana, map[string]any{"title": "Sin internet"})

	t.Run("un ticket nuevo avisa a todo el equipo", func(t *testing.T) {
		if got := notifKinds(e.notifications(luis)); !slices.Equal(got, []string{"new_ticket"}) {
			t.Errorf("luis = %v", got)
		}
		if got := e.notifications(ana); len(got.Items) != 0 {
			t.Errorf("la clienta no se avisa a sí misma: %+v", got.Items)
		}
		mails := e.outbox()
		if len(mails) != 1 || !strings.HasPrefix(mails[0], "luis@equipo.test | [#") {
			t.Errorf("correos = %v (marta los tiene desactivados)", mails)
		}
	})

	expectStatus(t, e.patch(ticketPath(tk.ID), marta.Token, map[string]any{"assignee_id": luis.ID}), http.StatusOK)
	t.Run("asignar avisa al agente", func(t *testing.T) {
		n := e.notifications(luis)
		if n.Items[0].Kind != "assigned" || n.Items[0].Actor.ID != marta.ID || n.Items[0].Title != "Sin internet" {
			t.Errorf("notificación = %+v", n.Items[0])
		}
		if n.Unread != 2 {
			t.Errorf("unread = %d", n.Unread)
		}
	})

	t.Run("la clienta responde: avisa al asignado", func(t *testing.T) {
		e.post(ticketPath(tk.ID)+"/comments", ana.Token, map[string]any{"body": "¿Novedades?"})
		if k := e.notifications(luis).Items[0].Kind; k != "comment" {
			t.Errorf("luis última = %s", k)
		}
	})

	t.Run("una mención avisa una sola vez aunque también sea el asignado", func(t *testing.T) {
		before := len(e.notifications(luis).Items)
		e.post(ticketPath(tk.ID)+"/comments", marta.Token, map[string]any{
			"body": "@luis ramírez mira esto, y @Martina no existe", "internal": true,
		})
		n := e.notifications(luis)
		if len(n.Items) != before+1 || n.Items[0].Kind != "mention" {
			t.Errorf("notificaciones de luis = %v", notifKinds(n))
		}
		if len(e.notifications(ana).Items) != 0 {
			t.Error("una nota interna no avisa a la clienta")
		}
	})

	t.Run("respuesta pública y estado avisan a la clienta", func(t *testing.T) {
		e.post(ticketPath(tk.ID)+"/comments", luis.Token, map[string]any{"body": "Reinicia el router"})
		e.patch(ticketPath(tk.ID), luis.Token, map[string]any{"status": "resolved"})
		got := notifKinds(e.notifications(ana))
		if !slices.Equal(got, []string{"status", "comment"}) {
			t.Errorf("ana = %v", got)
		}
		mails := strings.Join(e.outbox(), "\n")
		if !strings.Contains(mails, "ana@cliente.test | [#") || !strings.Contains(mails, "Tu ticket está resuelto") {
			t.Errorf("correos = %s", mails)
		}
	})

	t.Run("marcar como leídas", func(t *testing.T) {
		n := e.notifications(luis)
		expectStatus(t, e.post("/api/notifications/read", luis.Token, map[string]any{"ids": []int64{n.Items[0].ID}}), http.StatusNoContent)
		if got := e.notifications(luis).Unread; got != n.Unread-1 {
			t.Errorf("unread = %d, antes %d", got, n.Unread)
		}
		expectStatus(t, e.post("/api/notifications/read", luis.Token, map[string]any{}), http.StatusNoContent)
		if got := e.notifications(luis).Unread; got != 0 {
			t.Errorf("unread tras leer todas = %d", got)
		}
		// No se pueden tocar las de otro usuario.
		if e.notifications(ana).Unread == 0 {
			t.Error("las de ana no debían marcarse")
		}
	})

	t.Run("desactivar los correos desde el perfil", func(t *testing.T) {
		res := e.patch("/api/me", ana.Token, map[string]any{"email_notifications": false})
		expectStatus(t, res, http.StatusOK)
		if res.field("email_notifications") != false {
			t.Errorf("email_notifications = %v", res.field("email_notifications"))
		}
	})
}

func TestMentionMatching(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("cliente")
	e.createUser("Ana", "ana@equipo.test", "agent")
	anabel := e.createUser("Anabel", "anabel@equipo.test", "agent")
	jefe := e.agent("jefe")
	tk := e.newTicket(ana, map[string]any{"title": "x"})

	e.post(ticketPath(tk.ID)+"/comments", jefe.Token, map[string]any{"body": "Hola @Anabel.", "internal": true})
	var anaUser actor
	e.pool.QueryRow(context.Background(), `SELECT id FROM users WHERE email = 'ana@equipo.test'`).Scan(&anaUser.ID)
	token, _, _ := e.issuer.Issue(anaUser.ID, "agent")
	anaUser.Token = token

	if got := notifKinds(e.notifications(anabel)); !slices.Contains(got, "mention") {
		t.Errorf("anabel debió ser mencionada: %v", got)
	}
	if got := notifKinds(e.notifications(anaUser)); slices.Contains(got, "mention") {
		t.Errorf("@Anabel no menciona a Ana: %v", got)
	}
}

func TestPasswordReset(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")

	expectStatus(t, e.post("/api/auth/forgot", "", map[string]any{"email": "nadie@cliente.test"}), http.StatusAccepted)
	if len(e.outbox()) != 0 {
		t.Fatal("no debe enviarse correo a un email inexistente")
	}
	expectStatus(t, e.post("/api/auth/forgot", "", map[string]any{"email": "ANA@cliente.test"}), http.StatusAccepted)

	var body string
	e.pool.QueryRow(context.Background(), `SELECT body FROM email_outbox WHERE to_email = $1`, ana.Email).Scan(&body)
	m := regexp.MustCompile(`/reset-password\?token=([0-9a-f]+)`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("el correo no trae el enlace: %s", body)
	}
	token := m[1]

	expectFieldError(t, e.post("/api/auth/reset", "", map[string]any{"token": "malo", "password": "nueva12345"}), "token")
	expectFieldError(t, e.post("/api/auth/reset", "", map[string]any{"token": token, "password": "corta"}), "password")

	time.Sleep(1100 * time.Millisecond) // el token viejo es de un segundo anterior
	res := e.post("/api/auth/reset", "", map[string]any{"token": token, "password": "nueva12345"})
	expectStatus(t, res, http.StatusOK)
	if res.field("token") == "" {
		t.Error("debería iniciar sesión")
	}
	expectStatus(t, e.post("/api/auth/login", "", map[string]any{"email": ana.Email, "password": "nueva12345"}), http.StatusOK)
	expectStatus(t, e.get("/api/me", ana.Token), http.StatusUnauthorized)

	t.Run("el enlace solo sirve una vez", func(t *testing.T) {
		expectFieldError(t, e.post("/api/auth/reset", "", map[string]any{"token": token, "password": "otra123456"}), "token")
	})
	t.Run("los enlaces caducados no sirven", func(t *testing.T) {
		e.post("/api/auth/forgot", "", map[string]any{"email": ana.Email})
		mustExec(t, e, `UPDATE password_resets SET expires_at = now() - interval '1 minute' WHERE used_at IS NULL`)
		var body string
		e.pool.QueryRow(context.Background(), `SELECT body FROM email_outbox ORDER BY id DESC LIMIT 1`).Scan(&body)
		tok := regexp.MustCompile(`token=([0-9a-f]+)`).FindStringSubmatch(body)[1]
		expectFieldError(t, e.post("/api/auth/reset", "", map[string]any{"token": tok, "password": "otra123456"}), "token")
	})
}

func TestStream(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	tk := e.newTicket(ana, map[string]any{"title": "x"})

	srv := httptest.NewServer(e.handler)
	defer srv.Close()

	open := func(a actor) (*bufio.Reader, func()) {
		ctx, cancel := context.WithCancel(context.Background())
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/stream", nil)
		req.Header.Set("Authorization", "Bearer "+a.Token)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "text/event-stream" {
			t.Fatalf("stream: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
		}
		return bufio.NewReader(res.Body), func() { cancel(); res.Body.Close() }
	}
	next := func(r *bufio.Reader) string {
		t.Helper()
		done := make(chan string, 1)
		go func() {
			for {
				line, err := r.ReadString('\n')
				if err != nil {
					done <- "error: " + err.Error()
					return
				}
				if strings.HasPrefix(line, "data: ") {
					done <- strings.TrimSpace(strings.TrimPrefix(line, "data: "))
					return
				}
			}
		}()
		select {
		case s := <-done:
			return s
		case <-time.After(3 * time.Second):
			t.Fatal("no llegó ningún evento")
			return ""
		}
	}

	agentStream, closeAgent := open(luis)
	defer closeAgent()
	customerStream, closeCustomer := open(ana)
	defer closeCustomer()

	// Una nota interna solo llega al equipo; una respuesta pública también a la clienta.
	e.post(ticketPath(tk.ID)+"/comments", luis.Token, map[string]any{"body": "nota", "internal": true})
	if got := next(agentStream); !strings.Contains(got, `"type":"ticket"`) {
		t.Errorf("agente recibió %s", got)
	}
	e.post(ticketPath(tk.ID)+"/comments", luis.Token, map[string]any{"body": "Hola"})
	if got := next(agentStream); !strings.Contains(got, `"ticket_id":`+itoa(tk.ID)) {
		t.Errorf("agente recibió %s", got)
	}
	first := next(customerStream)
	if !strings.Contains(first, `"type":"ticket"`) {
		t.Errorf("la clienta debió recibir primero el ticket (no la nota interna): %s", first)
	}
	if got := next(customerStream); !strings.Contains(got, `"type":"notification"`) {
		t.Errorf("la clienta debió recibir la notificación: %s", got)
	}
}
