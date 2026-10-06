package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
)

type authResp struct {
	Token string   `json:"token"`
	User  api.User `json:"user"`
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	res := e.get("/api/health", "")
	expectStatus(t, res, http.StatusOK)
	if res.field("status") != "ok" {
		t.Errorf("respuesta inesperada: %s", res.Body)
	}
}

func TestRegisterThenLogin(t *testing.T) {
	e := newEnv(t)

	res := e.post("/api/auth/register", "", map[string]any{
		"name": "  Ana  ", "email": "Ana@Ejemplo.com", "password": testPassword,
	})
	expectStatus(t, res, http.StatusCreated)
	var reg authResp
	res.decode(&reg)
	if reg.Token == "" || reg.User.Role != "customer" || reg.User.Name != "Ana" {
		t.Fatalf("registro inesperado: %+v", reg)
	}
	if strings.Contains(string(res.Body), "password") {
		t.Error("la respuesta no debe incluir datos de la contraseña")
	}

	// El token recién emitido sirve.
	me := e.get("/api/me", reg.Token)
	expectStatus(t, me, http.StatusOK)

	// El login ignora mayúsculas y espacios en el email.
	login := e.post("/api/auth/login", "", map[string]any{
		"email": " ana@ejemplo.COM ", "password": testPassword,
	})
	expectStatus(t, login, http.StatusOK)
	var got authResp
	login.decode(&got)
	if got.User.ID != reg.User.ID {
		t.Errorf("el login devolvió otro usuario: %d != %d", got.User.ID, reg.User.ID)
	}
}

func TestLoginFailuresDoNotRevealWhichAccountsExist(t *testing.T) {
	e := newEnv(t)
	e.customer("ana")

	wrongPassword := e.post("/api/auth/login", "", map[string]any{"email": "ana@cliente.test", "password": "incorrecta"})
	unknownEmail := e.post("/api/auth/login", "", map[string]any{"email": "nadie@cliente.test", "password": "incorrecta"})

	expectStatus(t, wrongPassword, http.StatusUnauthorized)
	expectStatus(t, unknownEmail, http.StatusUnauthorized)
	if wrongPassword.errorText() != unknownEmail.errorText() {
		t.Errorf("mensajes distintos revelan qué cuentas existen: %q vs %q",
			wrongPassword.errorText(), unknownEmail.errorText())
	}
}

func TestRegisterValidation(t *testing.T) {
	e := newEnv(t)

	t.Run("campos inválidos", func(t *testing.T) {
		res := e.post("/api/auth/register", "", map[string]any{"name": "  ", "email": "nope", "password": "1"})
		expectStatus(t, res, http.StatusUnprocessableEntity)
		for _, f := range []string{"name", "email", "password"} {
			if _, ok := res.fieldErrors()[f]; !ok {
				t.Errorf("falta el error del campo %q: %s", f, res.Body)
			}
		}
	})

	t.Run("contraseña demasiado larga para bcrypt", func(t *testing.T) {
		res := e.post("/api/auth/register", "", map[string]any{
			"name": "Ana", "email": "ana@x.com", "password": strings.Repeat("a", 73),
		})
		expectFieldError(t, res, "password")
	})

	t.Run("no se puede elegir el rol", func(t *testing.T) {
		res := e.post("/api/auth/register", "", `{"name":"Mal","email":"mal@x.com","password":"secreto123","role":"admin"}`)
		expectStatus(t, res, http.StatusBadRequest)
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE email = 'mal@x.com'`).Scan(&n)
		if n != 0 {
			t.Error("no debería haberse creado el usuario")
		}
	})

	t.Run("JSON roto", func(t *testing.T) {
		expectStatus(t, e.post("/api/auth/register", "", `{"name":`), http.StatusBadRequest)
	})

	t.Run("email repetido sin distinguir mayúsculas", func(t *testing.T) {
		body := map[string]any{"name": "Ana", "email": "ana@x.com", "password": testPassword}
		expectStatus(t, e.post("/api/auth/register", "", body), http.StatusCreated)
		body["email"] = "ANA@x.com"
		res := e.post("/api/auth/register", "", body)
		expectStatus(t, res, http.StatusConflict)
	})
}

func TestAuthenticationRequired(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")

	for _, path := range []string{"/api/me", "/api/tickets", "/api/tickets/1", "/api/tickets/1/comments", "/api/stats", "/api/users"} {
		if res := e.get(path, ""); res.Status != http.StatusUnauthorized {
			t.Errorf("GET %s sin token: %d, se esperaba 401", path, res.Status)
		}
	}
	if res := e.post("/api/tickets", "", map[string]any{"title": "x"}); res.Status != http.StatusUnauthorized {
		t.Errorf("POST /api/tickets sin token: %d", res.Status)
	}

	t.Run("token basura", func(t *testing.T) {
		expectStatus(t, e.get("/api/me", "no.es.un.token"), http.StatusUnauthorized)
	})

	t.Run("token firmado con otra clave", func(t *testing.T) {
		other := auth.NewIssuer([]byte("ffffffffffffffffffffffffffffffff"), time.Hour)
		token, _, _ := other.Issue(ana.ID, auth.RoleAdmin)
		expectStatus(t, e.get("/api/me", token), http.StatusUnauthorized)
	})

	t.Run("token expirado", func(t *testing.T) {
		expired := auth.NewIssuer([]byte(testSecret), -time.Minute)
		token, _, _ := expired.Issue(ana.ID, auth.RoleCustomer)
		expectStatus(t, e.get("/api/me", token), http.StatusUnauthorized)
	})

	t.Run("usuario eliminado", func(t *testing.T) {
		gone := e.customer("fantasma")
		if _, err := e.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, gone.ID); err != nil {
			t.Fatal(err)
		}
		expectStatus(t, e.get("/api/me", gone.Token), http.StatusUnauthorized)
	})
}

func TestRoleInTokenIsNotTrusted(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")

	// Un token que dice "admin" para un usuario que en la base es cliente no da acceso.
	token, _, _ := e.issuer.Issue(ana.ID, auth.RoleAdmin)
	expectStatus(t, e.get("/api/stats", token), http.StatusForbidden)
}

func TestRoleChangeAppliesImmediately(t *testing.T) {
	e := newEnv(t)
	luis := e.agent("luis")

	expectStatus(t, e.get("/api/stats", luis.Token), http.StatusOK)

	e.setRole(luis.ID, "customer") // el mismo token, ya emitido como agente
	expectStatus(t, e.get("/api/stats", luis.Token), http.StatusForbidden)
}

func TestListUsers(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	root := e.admin("root")

	expectStatus(t, e.get("/api/users", ana.Token), http.StatusForbidden)

	type users struct {
		Items []api.User `json:"items"`
	}
	var all, agents users
	res := e.get("/api/users", luis.Token)
	expectStatus(t, res, http.StatusOK)
	res.decode(&all)
	if len(all.Items) != 3 {
		t.Errorf("usuarios = %d, se esperaban 3", len(all.Items))
	}

	res = e.get("/api/users?role=agent", root.Token)
	expectStatus(t, res, http.StatusOK)
	res.decode(&agents)
	if len(agents.Items) != 1 || agents.Items[0].ID != luis.ID {
		t.Errorf("filtro por rol: %+v", agents.Items)
	}

	expectStatus(t, e.get("/api/users?role=reina", root.Token), http.StatusBadRequest)
}

func TestChangeRole(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	luis := e.agent("luis")
	root := e.admin("root")
	path := "/api/users/" + itoa(ana.ID) + "/role"

	t.Run("solo administradores", func(t *testing.T) {
		expectStatus(t, e.patch(path, ana.Token, map[string]any{"role": "admin"}), http.StatusForbidden)
		expectStatus(t, e.patch(path, luis.Token, map[string]any{"role": "agent"}), http.StatusForbidden)
	})

	t.Run("un administrador promueve a un cliente", func(t *testing.T) {
		res := e.patch(path, root.Token, map[string]any{"role": "agent"})
		expectStatus(t, res, http.StatusOK)
		if res.field("role") != "agent" {
			t.Errorf("rol = %v", res.field("role"))
		}
		// El cambio vale ya para el token que Ana tenía.
		expectStatus(t, e.get("/api/stats", ana.Token), http.StatusOK)
	})

	t.Run("rol inválido", func(t *testing.T) {
		expectStatus(t, e.patch(path, root.Token, map[string]any{"role": "reina"}), http.StatusUnprocessableEntity)
	})

	t.Run("usuario que no existe", func(t *testing.T) {
		res := e.patch("/api/users/999999/role", root.Token, map[string]any{"role": "agent"})
		expectStatus(t, res, http.StatusNotFound)
	})

	t.Run("id inválido", func(t *testing.T) {
		expectStatus(t, e.patch("/api/users/abc/role", root.Token, map[string]any{"role": "agent"}), http.StatusBadRequest)
	})

	t.Run("un administrador no puede quitarse el rol", func(t *testing.T) {
		res := e.patch("/api/users/"+itoa(root.ID)+"/role", root.Token, map[string]any{"role": "customer"})
		expectStatus(t, res, http.StatusUnprocessableEntity)
		expectStatus(t, e.get("/api/stats", root.Token), http.StatusOK)
	})
}

func TestCORS(t *testing.T) {
	e := newEnv(t)

	allowed := e.do(http.MethodOptions, "/api/tickets", "", nil, withHeader("Origin", "http://localhost:5173"))
	expectStatus(t, allowed, http.StatusNoContent)
	if got := allowed.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("origen permitido: %q", got)
	}

	denied := e.do(http.MethodOptions, "/api/tickets", "", nil, withHeader("Origin", "https://malo.example"))
	if got := denied.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("un origen no permitido no debe recibir cabecera CORS, got %q", got)
	}
}
