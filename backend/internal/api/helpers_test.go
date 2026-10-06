package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
	"github.com/MADGRISMAD/MiColmena/backend/internal/testutil"
)

const (
	testSecret   = "0123456789abcdef0123456789abcdef"
	testPassword = "secreto123"
)

var (
	hashOnce sync.Once
	testHash string
)

// passwordHash calcula una sola vez el hash de testPassword: bcrypt es lento a propósito
// y las pruebas crean muchos usuarios directamente en la base.
func passwordHash(t testing.TB) string {
	t.Helper()
	hashOnce.Do(func() {
		h, err := auth.HashPassword(testPassword)
		if err != nil {
			t.Fatal(err)
		}
		testHash = h
	})
	return testHash
}

// env es una API completa con su propio esquema de PostgreSQL.
type env struct {
	t       testing.TB
	pool    *pgxpool.Pool
	handler http.Handler
	issuer  *auth.Issuer
}

// newEnv crea el entorno. Por defecto el límite de intentos es muy alto para no estorbar
// a las demás pruebas; las que lo prueban pasan sus propias opciones.
func newEnv(t testing.TB, mutate ...func(*api.Options)) *env {
	t.Helper()
	pool := testutil.NewPool(t)
	issuer := auth.NewIssuer([]byte(testSecret), time.Hour)
	opts := api.Options{
		CORSOrigins:    []string{"http://localhost:5173"},
		AuthRatePerMin: 100_000,
		UploadDir:      t.TempDir(),
		MaxUploadBytes: 1 << 20,
	}
	for _, m := range mutate {
		m(&opts)
	}
	return &env{t: t, pool: pool, handler: api.NewServer(pool, issuer, opts).Handler(), issuer: issuer}
}

// response es una respuesta HTTP ya leída.
type response struct {
	t      testing.TB
	Status int
	Header http.Header
	Body   []byte
}

func (r response) decode(v any) {
	r.t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		r.t.Fatalf("respuesta no es JSON válido (%d): %s", r.Status, r.Body)
	}
}

// field lee un campo de primer nivel de una respuesta JSON de objeto.
func (r response) field(name string) any {
	r.t.Helper()
	var m map[string]any
	r.decode(&m)
	return m[name]
}

func (r response) errorText() string {
	s, _ := r.field("error").(string)
	return s
}

// fieldErrors devuelve los errores por campo de una respuesta 422.
func (r response) fieldErrors() map[string]any {
	r.t.Helper()
	m, _ := r.field("fields").(map[string]any)
	return m
}

type reqOpt func(*http.Request)

func withRemoteAddr(addr string) reqOpt { return func(r *http.Request) { r.RemoteAddr = addr } }
func withHeader(k, v string) reqOpt     { return func(r *http.Request) { r.Header.Set(k, v) } }

// do ejecuta una petición contra el manejador sin pasar por la red.
func (e *env) do(method, path, token string, body any, opts ...reqOpt) response {
	e.t.Helper()
	var reader *bytes.Reader
	switch b := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(b))
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			e.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, reader)
	req.RemoteAddr = "192.0.2.10:4321"
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for _, o := range opts {
		o(req)
	}

	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return response{t: e.t, Status: rec.Code, Header: rec.Header(), Body: rec.Body.Bytes()}
}

func (e *env) get(path, token string, opts ...reqOpt) response {
	e.t.Helper()
	return e.do(http.MethodGet, path, token, nil, opts...)
}
func (e *env) post(path, token string, body any, opts ...reqOpt) response {
	e.t.Helper()
	return e.do(http.MethodPost, path, token, body, opts...)
}
func (e *env) patch(path, token string, body any, opts ...reqOpt) response {
	e.t.Helper()
	return e.do(http.MethodPatch, path, token, body, opts...)
}

// actor es un usuario ya creado y con sesión iniciada.
type actor struct {
	ID    int64
	Name  string
	Email string
	Token string
}

// createUser inserta un usuario directamente en la base (sin pasar por bcrypt) con la contraseña testPassword.
func (e *env) createUser(name, email, role string) actor {
	e.t.Helper()
	var id int64
	err := e.pool.QueryRow(context.Background(), `
		INSERT INTO users (name, email, password_hash, role) VALUES ($1, $2, $3, $4) RETURNING id`,
		name, email, passwordHash(e.t), role).Scan(&id)
	if err != nil {
		e.t.Fatalf("crear usuario %s: %v", email, err)
	}
	token, _, err := e.issuer.Issue(id, role)
	if err != nil {
		e.t.Fatal(err)
	}
	return actor{ID: id, Name: name, Email: email, Token: token}
}

func (e *env) customer(name string) actor {
	e.t.Helper()
	return e.createUser(name, name+"@cliente.test", auth.RoleCustomer)
}
func (e *env) agent(name string) actor {
	e.t.Helper()
	return e.createUser(name, name+"@equipo.test", auth.RoleAgent)
}
func (e *env) admin(name string) actor {
	e.t.Helper()
	return e.createUser(name, name+"@equipo.test", auth.RoleAdmin)
}

func (e *env) setRole(userID int64, role string) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), `UPDATE users SET role = $2 WHERE id = $1`, userID, role); err != nil {
		e.t.Fatal(err)
	}
}

// newTicket crea un ticket por la API y devuelve su id.
func (e *env) newTicket(a actor, body map[string]any) api.Ticket {
	e.t.Helper()
	if _, ok := body["title"]; !ok {
		body["title"] = "Ticket de prueba"
	}
	res := e.post("/api/tickets", a.Token, body)
	if res.Status != http.StatusCreated {
		e.t.Fatalf("crear ticket: %d %s", res.Status, res.Body)
	}
	var tk api.Ticket
	res.decode(&tk)
	return tk
}

func (e *env) ticket(a actor, id int64) api.Ticket {
	e.t.Helper()
	res := e.get(ticketPath(id), a.Token)
	if res.Status != http.StatusOK {
		e.t.Fatalf("leer ticket %d: %d %s", id, res.Status, res.Body)
	}
	var tk api.Ticket
	res.decode(&tk)
	return tk
}

func ticketPath(id int64) string { return "/api/tickets/" + itoa(id) }

func itoa(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// expectStatus falla si la respuesta no tiene el código esperado.
func expectStatus(t testing.TB, res response, want int) {
	t.Helper()
	if res.Status != want {
		t.Fatalf("código %d, se esperaba %d: %s", res.Status, want, res.Body)
	}
}

// expectFieldError falla si la respuesta no es un 422 con error en el campo indicado.
func expectFieldError(t testing.TB, res response, field string) {
	t.Helper()
	expectStatus(t, res, http.StatusUnprocessableEntity)
	if _, ok := res.fieldErrors()[field]; !ok {
		t.Fatalf("se esperaba error en el campo %q, got %s", field, res.Body)
	}
}

func ids(tickets []api.Ticket) []int64 {
	out := make([]int64, len(tickets))
	for i, tk := range tickets {
		out[i] = tk.ID
	}
	return out
}

type ticketPage struct {
	Items      []api.Ticket `json:"items"`
	NextCursor *int64       `json:"next_cursor"`
}

func (e *env) list(a actor, query string) ticketPage {
	e.t.Helper()
	res := e.get("/api/tickets"+query, a.Token)
	if res.Status != http.StatusOK {
		e.t.Fatalf("listar tickets%s: %d %s", query, res.Status, res.Body)
	}
	var p ticketPage
	res.decode(&p)
	return p
}

func mustExec(t testing.TB, e *env, sql string, args ...any) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}
