package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
)

func TestAdminCreatesAndManagesUsers(t *testing.T) {
	e := newEnv(t)
	boss := e.admin("jefa")
	luis := e.agent("luis")

	t.Run("solo un administrador crea usuarios", func(t *testing.T) {
		res := e.post("/api/users", luis.Token, map[string]any{
			"name": "Eva", "email": "eva@equipo.test", "role": "agent", "password": "inicial123",
		})
		expectStatus(t, res, http.StatusForbidden)
	})

	res := e.post("/api/users", boss.Token, map[string]any{
		"name": "Eva", "email": "eva@equipo.test", "role": "agent", "password": "inicial123",
	})
	expectStatus(t, res, http.StatusCreated)
	evaID := int64(res.field("id").(float64))

	t.Run("el nuevo agente puede entrar", func(t *testing.T) {
		res := e.post("/api/auth/login", "", map[string]any{"email": "eva@equipo.test", "password": "inicial123"})
		expectStatus(t, res, http.StatusOK)
	})

	t.Run("email repetido", func(t *testing.T) {
		res := e.post("/api/users", boss.Token, map[string]any{
			"name": "Otra", "email": "EVA@equipo.test", "role": "agent", "password": "inicial123",
		})
		expectFieldError(t, res, "email")
	})

	t.Run("validación", func(t *testing.T) {
		res := e.post("/api/users", boss.Token, map[string]any{
			"name": "", "email": "no-es-email", "role": "jefe", "password": "corta",
		})
		expectStatus(t, res, http.StatusUnprocessableEntity)
		for _, f := range []string{"name", "email", "role", "password"} {
			if _, ok := res.fieldErrors()[f]; !ok {
				t.Errorf("falta error en %s: %s", f, res.Body)
			}
		}
	})

	t.Run("el administrador no puede desactivarse ni degradarse", func(t *testing.T) {
		expectFieldError(t, e.patch("/api/users/"+itoa(boss.ID), boss.Token, map[string]any{"active": false}), "active")
		expectFieldError(t, e.patch("/api/users/"+itoa(boss.ID), boss.Token, map[string]any{"role": "agent"}), "role")
	})

	t.Run("cambiar nombre y contraseña", func(t *testing.T) {
		res := e.patch("/api/users/"+itoa(evaID), boss.Token, map[string]any{"name": "Eva Ruiz", "password": "nueva12345"})
		expectStatus(t, res, http.StatusOK)
		if res.field("name") != "Eva Ruiz" {
			t.Errorf("nombre = %v", res.field("name"))
		}
		expectStatus(t, e.post("/api/auth/login", "", map[string]any{"email": "eva@equipo.test", "password": "nueva12345"}), http.StatusOK)
	})

	t.Run("usuario inexistente", func(t *testing.T) {
		expectStatus(t, e.patch("/api/users/999999", boss.Token, map[string]any{"name": "X"}), http.StatusNotFound)
	})
}

func TestDeactivatedUserIsLockedOutAndReleasesTickets(t *testing.T) {
	e := newEnv(t)
	boss := e.admin("jefa")
	luis := e.agent("luis")
	ana := e.customer("ana")

	pending := e.newTicket(ana, map[string]any{"title": "Pendiente"})
	done := e.newTicket(ana, map[string]any{"title": "Resuelto"})
	expectStatus(t, e.patch(ticketPath(pending.ID), luis.Token, map[string]any{"assignee_id": luis.ID}), http.StatusOK)
	expectStatus(t, e.patch(ticketPath(done.ID), luis.Token, map[string]any{"assignee_id": luis.ID, "status": "resolved"}), http.StatusOK)

	res := e.patch("/api/users/"+itoa(luis.ID), boss.Token, map[string]any{"active": false})
	expectStatus(t, res, http.StatusOK)
	if res.field("active") != false {
		t.Fatalf("active = %v", res.field("active"))
	}

	t.Run("su token deja de valer", func(t *testing.T) {
		expectStatus(t, e.get("/api/me", luis.Token), http.StatusUnauthorized)
	})
	t.Run("no puede iniciar sesión", func(t *testing.T) {
		res := e.post("/api/auth/login", "", map[string]any{"email": luis.Email, "password": testPassword})
		expectStatus(t, res, http.StatusForbidden)
	})
	t.Run("sus tickets pendientes quedan sin asignar", func(t *testing.T) {
		if got := e.ticket(boss, pending.ID); got.Assignee != nil {
			t.Errorf("pendiente sigue asignado: %+v", got.Assignee)
		}
		if got := e.ticket(boss, done.ID); got.Assignee == nil || got.Assignee.ID != luis.ID {
			t.Errorf("el resuelto debería conservar su agente: %+v", got.Assignee)
		}
	})
	t.Run("no se le pueden asignar tickets", func(t *testing.T) {
		expectFieldError(t, e.patch(ticketPath(pending.ID), boss.Token, map[string]any{"assignee_id": luis.ID}), "assignee_id")
	})
	t.Run("la lista oculta a los desactivados salvo con active=all", func(t *testing.T) {
		var list struct {
			Items []struct {
				ID int64 `json:"id"`
			} `json:"items"`
		}
		e.get("/api/users", boss.Token).decode(&list)
		for _, u := range list.Items {
			if u.ID == luis.ID {
				t.Error("aparece un usuario desactivado")
			}
		}
		e.get("/api/users?active=all", boss.Token).decode(&list)
		found := false
		for _, u := range list.Items {
			found = found || u.ID == luis.ID
		}
		if !found {
			t.Error("active=all debería incluirlo")
		}
	})
	t.Run("al reactivarlo puede volver a entrar", func(t *testing.T) {
		expectStatus(t, e.patch("/api/users/"+itoa(luis.ID), boss.Token, map[string]any{"active": true}), http.StatusOK)
		expectStatus(t, e.post("/api/auth/login", "", map[string]any{"email": luis.Email, "password": testPassword}), http.StatusOK)
	})
}

func TestUserSearch(t *testing.T) {
	e := newEnv(t)
	boss := e.admin("jefa")
	e.customer("marta")
	e.customer("pedro")

	var list struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	e.get("/api/users?q=mar", boss.Token).decode(&list)
	if len(list.Items) != 1 || list.Items[0].Name != "marta" {
		t.Errorf("búsqueda = %+v", list.Items)
	}
	// Los comodines de LIKE se buscan como texto.
	e.get("/api/users?q=%25", boss.Token).decode(&list)
	if len(list.Items) != 0 {
		t.Errorf("%% no debería coincidir con todo: %+v", list.Items)
	}
}

func TestProfile(t *testing.T) {
	e := newEnv(t)
	ana := e.customer("ana")
	e.customer("berta")

	t.Run("cambiar el nombre no pide contraseña", func(t *testing.T) {
		res := e.patch("/api/me", ana.Token, map[string]any{"name": "Ana López"})
		expectStatus(t, res, http.StatusOK)
		if res.field("name") != "Ana López" {
			t.Errorf("nombre = %v", res.field("name"))
		}
	})

	t.Run("cambiar el email pide la contraseña actual", func(t *testing.T) {
		expectFieldError(t, e.patch("/api/me", ana.Token, map[string]any{"email": "ana@nuevo.test"}), "current_password")
		expectFieldError(t, e.patch("/api/me", ana.Token, map[string]any{"email": "berta@cliente.test", "current_password": testPassword}), "email")
		res := e.patch("/api/me", ana.Token, map[string]any{"email": "ana@nuevo.test", "current_password": testPassword})
		expectStatus(t, res, http.StatusOK)
		if res.field("email") != "ana@nuevo.test" {
			t.Errorf("email = %v", res.field("email"))
		}
	})

	t.Run("cambiar la contraseña cierra las demás sesiones", func(t *testing.T) {
		expectFieldError(t, e.post("/api/me/password", ana.Token, map[string]any{
			"current_password": "mala", "new_password": "otra12345",
		}), "current_password")
		expectFieldError(t, e.post("/api/me/password", ana.Token, map[string]any{
			"current_password": testPassword, "new_password": "corta",
		}), "new_password")

		// El token anterior se emitió en un segundo anterior al cambio.
		time.Sleep(1100 * time.Millisecond)
		res := e.post("/api/me/password", ana.Token, map[string]any{
			"current_password": testPassword, "new_password": "otra12345",
		})
		expectStatus(t, res, http.StatusOK)
		newToken, _ := res.field("token").(string)

		expectStatus(t, e.get("/api/me", ana.Token), http.StatusUnauthorized)
		expectStatus(t, e.get("/api/me", newToken), http.StatusOK)
		expectStatus(t, e.post("/api/auth/login", "", map[string]any{"email": "ana@nuevo.test", "password": "otra12345"}), http.StatusOK)
	})
}

func TestPermanentAdmins(t *testing.T) {
	e := newEnv(t)
	boss := e.admin("jefa")
	mad := e.createUser("Mad", "MadGrisMad@gmail.com", auth.RoleAdmin)
	luis := e.createUser("Luis", "luispantoja1102@gmail.com", auth.RoleAdmin)
	impostor := e.createUser("Falsa", "mayra.bamaca09@gmail.com", auth.RoleCustomer)
	path := "/api/users/" + itoa(mad.ID)

	t.Run("nadie los baja de rol, los desactiva ni les cambia el email", func(t *testing.T) {
		expectStatus(t, e.patch(path, boss.Token, map[string]any{"role": "agent"}), http.StatusForbidden)
		expectStatus(t, e.patch(path+"/role", boss.Token, map[string]any{"role": "customer"}), http.StatusForbidden)
		expectStatus(t, e.patch(path, boss.Token, map[string]any{"active": false}), http.StatusForbidden)
		expectStatus(t, e.patch(path, boss.Token, map[string]any{"email": "otro@equipo.test"}), http.StatusForbidden)
		expectStatus(t, e.patch(path, luis.Token, map[string]any{"active": false}), http.StatusForbidden)
		expectStatus(t, e.patch(path, boss.Token, map[string]any{"name": "Mad G."}), http.StatusOK)
	})

	t.Run("su contraseña solo la cambia otro permanente", func(t *testing.T) {
		expectStatus(t, e.patch(path, boss.Token, map[string]any{"password": "nueva12345"}), http.StatusForbidden)
		expectStatus(t, e.patch(path, luis.Token, map[string]any{"password": "nueva12345"}), http.StatusOK)
	})

	t.Run("la lista marca a los permanentes", func(t *testing.T) {
		var list struct {
			Items []struct {
				ID        int64 `json:"id"`
				Permanent bool  `json:"permanent"`
			} `json:"items"`
		}
		e.get("/api/users?active=all", boss.Token).decode(&list)
		for _, u := range list.Items {
			if want := u.ID == mad.ID || u.ID == luis.ID; u.Permanent != want {
				t.Errorf("usuario %d: permanent = %v", u.ID, u.Permanent)
			}
		}
	})

	t.Run("un cliente con ese correo no queda protegido", func(t *testing.T) {
		expectStatus(t, e.patch("/api/users/"+itoa(impostor.ID), boss.Token, map[string]any{"active": false}), http.StatusOK)
	})
}
