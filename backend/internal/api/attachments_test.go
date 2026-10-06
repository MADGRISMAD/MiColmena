package api_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
)

// upload envía un archivo como multipart. commentID vacío = adjunto de la descripción.
func (e *env) upload(a actor, ticketID int64, commentID, filename string, content []byte) response {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if commentID != "" {
		mw.WriteField("comment_id", commentID)
	}
	if filename != "" {
		fw, _ := mw.CreateFormFile("file", filename)
		fw.Write(content)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, ticketPath(ticketID)+"/attachments", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+a.Token)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return response{t: e.t, Status: rec.Code, Header: rec.Header(), Body: rec.Body.Bytes()}
}

var pngHeader = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

func TestAttachments(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	otro := e.customer("otro")
	tk := e.newTicket(ana, map[string]any{"title": "Con captura"})

	res := e.upload(ana, tk.ID, "", "../../captura.png", pngHeader)
	expectStatus(t, res, http.StatusCreated)
	var shot api.Attachment
	res.decode(&shot)
	if shot.Filename != "captura.png" || shot.ContentType != "image/png" || shot.Size != int64(len(pngHeader)) || shot.CommentID != nil {
		t.Errorf("adjunto = %+v", shot)
	}

	t.Run("descarga con el tipo real y como adjunto", func(t *testing.T) {
		res := e.get("/api/attachments/"+itoa(shot.ID), ana.Token)
		expectStatus(t, res, http.StatusOK)
		if !bytes.Equal(res.Body, pngHeader) {
			t.Error("contenido distinto")
		}
		if res.Header.Get("Content-Type") != "image/png" || !strings.HasPrefix(res.Header.Get("Content-Disposition"), "attachment") {
			t.Errorf("cabeceras = %v", res.Header)
		}
	})

	t.Run("un HTML se sirve como binario genérico", func(t *testing.T) {
		res := e.upload(ana, tk.ID, "", "x.html", []byte("<html><script>alert(1)</script></html>"))
		expectStatus(t, res, http.StatusCreated)
		var a api.Attachment
		res.decode(&a)
		dl := e.get("/api/attachments/"+itoa(a.ID), ana.Token)
		if ct := dl.Header.Get("Content-Type"); ct != "application/octet-stream" {
			t.Errorf("Content-Type = %s", ct)
		}
		if dl.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Error("falta nosniff")
		}
	})

	t.Run("otro cliente no puede subir ni descargar", func(t *testing.T) {
		expectStatus(t, e.upload(otro, tk.ID, "", "a.txt", []byte("hola")), http.StatusNotFound)
		expectStatus(t, e.get("/api/attachments/"+itoa(shot.ID), otro.Token), http.StatusNotFound)
	})

	t.Run("adjuntos de una nota interna no los ve el cliente", func(t *testing.T) {
		res := e.post(ticketPath(tk.ID)+"/comments", luis.Token, map[string]any{"body": "log", "internal": true})
		expectStatus(t, res, http.StatusCreated)
		noteID := itoa(int64(res.field("id").(float64)))
		res = e.upload(luis, tk.ID, noteID, "servidor.log", []byte("error 500"))
		expectStatus(t, res, http.StatusCreated)
		var log api.Attachment
		res.decode(&log)
		if !log.Internal {
			t.Error("debería ser interno")
		}
		expectStatus(t, e.get("/api/attachments/"+itoa(log.ID), ana.Token), http.StatusNotFound)
		expectStatus(t, e.get("/api/attachments/"+itoa(log.ID), luis.Token), http.StatusOK)

		var list struct {
			Items []api.Attachment `json:"items"`
		}
		e.get(ticketPath(tk.ID)+"/attachments", ana.Token).decode(&list)
		for _, a := range list.Items {
			if a.Internal {
				t.Errorf("el cliente ve un adjunto interno: %+v", a)
			}
		}
		e.get(ticketPath(tk.ID)+"/attachments", luis.Token).decode(&list)
		if len(list.Items) != 3 {
			t.Errorf("el agente debería ver 3 adjuntos, ve %d", len(list.Items))
		}

		t.Run("no se puede adjuntar al comentario de otro", func(t *testing.T) {
			expectFieldError(t, e.upload(ana, tk.ID, noteID, "a.txt", []byte("x")), "comment_id")
		})
	})

	t.Run("validación", func(t *testing.T) {
		expectFieldError(t, e.upload(ana, tk.ID, "", "", nil), "file")
		expectFieldError(t, e.upload(ana, tk.ID, "", "vacío.txt", []byte{}), "file")
		big := bytes.Repeat([]byte("a"), (1<<20)+1)
		expectStatus(t, e.upload(ana, tk.ID, "", "grande.bin", big), http.StatusRequestEntityTooLarge)
	})

	t.Run("borrar: solo quien lo subió o un administrador", func(t *testing.T) {
		expectStatus(t, e.do(http.MethodDelete, "/api/attachments/"+itoa(shot.ID), luis.Token, nil), http.StatusForbidden)
		expectStatus(t, e.do(http.MethodDelete, "/api/attachments/"+itoa(shot.ID), ana.Token, nil), http.StatusNoContent)
		expectStatus(t, e.get("/api/attachments/"+itoa(shot.ID), ana.Token), http.StatusNotFound)
	})
}

func TestSLA(t *testing.T) {
	e := newEnv(t)
	boss := e.admin("jefa")
	luis := e.agent("luis")
	ana := e.customer("ana")

	tk := e.newTicket(ana, map[string]any{"title": "Urgente", "priority": "urgent"})
	if d := tk.FirstResponseDue.Sub(tk.CreatedAt); d.Minutes() != 60 {
		t.Errorf("plazo de primera respuesta urgente = %v", d)
	}
	if d := tk.ResolutionDue.Sub(tk.CreatedAt); d.Hours() != 4 {
		t.Errorf("plazo de resolución urgente = %v", d)
	}

	t.Run("la nota interna no cuenta como primera respuesta; la pública sí", func(t *testing.T) {
		e.post(ticketPath(tk.ID)+"/comments", luis.Token, map[string]any{"body": "mirando", "internal": true})
		if e.ticket(luis, tk.ID).FirstResponseAt != nil {
			t.Fatal("una nota interna no es respuesta")
		}
		e.post(ticketPath(tk.ID)+"/comments", ana.Token, map[string]any{"body": "¿hola?"})
		if e.ticket(luis, tk.ID).FirstResponseAt != nil {
			t.Fatal("el cliente no responde por el agente")
		}
		e.post(ticketPath(tk.ID)+"/comments", luis.Token, map[string]any{"body": "Ya lo vemos"})
		if e.ticket(luis, tk.ID).FirstResponseAt == nil {
			t.Fatal("falta first_response_at")
		}
	})

	t.Run("vencidos", func(t *testing.T) {
		late := e.newTicket(ana, map[string]any{"title": "Viejo", "priority": "high"})
		mustExec(t, e, `UPDATE tickets SET created_at = now() - interval '5 hours' WHERE id = $1`, late.ID)
		done := e.newTicket(ana, map[string]any{"title": "Viejo pero resuelto"})
		mustExec(t, e, `UPDATE tickets SET created_at = now() - interval '9 days', status = 'resolved' WHERE id = $1`, done.ID)

		if got := ids(e.list(luis, "?sla=breached").Items); len(got) != 1 || got[0] != late.ID {
			t.Errorf("sla=breached → %v", got)
		}
		var stats struct {
			SLABreached int `json:"sla_breached"`
		}
		e.get("/api/stats", luis.Token).decode(&stats)
		if stats.SLABreached != 1 {
			t.Errorf("stats.sla_breached = %d", stats.SLABreached)
		}
		expectStatus(t, e.get("/api/tickets?sla=raro", luis.Token), http.StatusBadRequest)
	})

	t.Run("el administrador cambia los plazos", func(t *testing.T) {
		body := map[string]any{"items": []map[string]any{{"priority": "urgent", "first_response_minutes": 30, "resolution_minutes": 120}}}
		expectStatus(t, e.do(http.MethodPut, "/api/sla", luis.Token, body), http.StatusForbidden)
		expectStatus(t, e.do(http.MethodPut, "/api/sla", boss.Token, body), http.StatusOK)
		if d := e.ticket(luis, tk.ID).FirstResponseDue.Sub(tk.CreatedAt); d.Minutes() != 30 {
			t.Errorf("nuevo plazo = %v", d)
		}
		bad := map[string]any{"items": []map[string]any{{"priority": "urgent", "first_response_minutes": 60, "resolution_minutes": 30}}}
		expectStatus(t, e.do(http.MethodPut, "/api/sla", boss.Token, bad), http.StatusUnprocessableEntity)
		bad = map[string]any{"items": []map[string]any{{"priority": "nada", "first_response_minutes": 1, "resolution_minutes": 2}}}
		expectStatus(t, e.do(http.MethodPut, "/api/sla", boss.Token, bad), http.StatusUnprocessableEntity)
	})
}

func TestSatisfaction(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	tk := e.newTicket(ana, map[string]any{"title": "x"})
	path := ticketPath(tk.ID) + "/satisfaction"

	expectFieldError(t, e.post(path, ana.Token, map[string]any{"rating": "good"}), "rating")
	expectStatus(t, e.patch(ticketPath(tk.ID), luis.Token, map[string]any{"status": "resolved"}), http.StatusOK)
	expectStatus(t, e.post(path, luis.Token, map[string]any{"rating": "good"}), http.StatusForbidden)
	expectFieldError(t, e.post(path, ana.Token, map[string]any{"rating": "regular"}), "rating")

	res := e.post(path, ana.Token, map[string]any{"rating": "bad", "comment": " Tardó mucho "})
	expectStatus(t, res, http.StatusOK)
	var got api.Ticket
	res.decode(&got)
	if got.Satisfaction != "bad" || got.SatisfactionComment != "Tardó mucho" || got.RatedAt == nil {
		t.Errorf("valoración = %+v", got)
	}
	expectStatus(t, e.post(path, ana.Token, map[string]any{"rating": "good"}), http.StatusOK)

	var stats struct {
		Satisfaction struct{ Good, Bad int } `json:"satisfaction_30d"`
	}
	e.get("/api/stats", luis.Token).decode(&stats)
	if stats.Satisfaction.Good != 1 || stats.Satisfaction.Bad != 0 {
		t.Errorf("stats = %+v", stats.Satisfaction)
	}
	evs := kinds(e.events(luis, tk.ID))
	if n := strings.Count(strings.Join(evs, ","), "satisfaction"); n != 2 {
		t.Errorf("historial = %v", evs)
	}
}
