package api_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/MADGRISMAD/MiColmena/backend/internal/api"
)

func loginAttempt(e *env, email, password string, opts ...reqOpt) response {
	e.t.Helper()
	return e.post("/api/auth/login", "", map[string]any{"email": email, "password": password}, opts...)
}

func TestAuthRateLimitPerIP(t *testing.T) {
	e := newEnv(t, func(o *api.Options) { o.AuthRatePerMin = 10 })

	// La ficha se repone cada 6 s, así que los intentos deben ser rápidos: registros vacíos
	// (comparten el límite y fallan en la validación, sin bcrypt) y un solo login.
	for i := 1; i <= 9; i++ {
		res := e.post("/api/auth/register", "", map[string]any{})
		if res.Status != http.StatusUnprocessableEntity {
			t.Fatalf("intento %d: %d, se esperaba 422", i, res.Status)
		}
	}
	expectStatus(t, loginAttempt(e, "u10@x.com", "incorrecta"), http.StatusUnauthorized)

	res := loginAttempt(e, "u11@x.com", "incorrecta")
	expectStatus(t, res, http.StatusTooManyRequests)
	retry, err := strconv.Atoi(res.Header.Get("Retry-After"))
	if err != nil || retry < 1 || retry > 60 {
		t.Errorf("Retry-After = %q, se esperaban segundos entre 1 y 60", res.Header.Get("Retry-After"))
	}
	if res.errorText() == "" {
		t.Error("el 429 debe explicar el motivo")
	}

	t.Run("otra IP no se ve afectada", func(t *testing.T) {
		res := loginAttempt(e, "u1@x.com", "incorrecta", withRemoteAddr("198.51.100.7:1111"))
		expectStatus(t, res, http.StatusUnauthorized)
	})

	t.Run("el registro comparte el límite con el login", func(t *testing.T) {
		res := e.post("/api/auth/register", "", map[string]any{"name": "Ana", "email": "ana@x.com", "password": testPassword})
		expectStatus(t, res, http.StatusTooManyRequests)
	})

	t.Run("el resto de la API no está limitado", func(t *testing.T) {
		expectStatus(t, e.get("/api/health", ""), http.StatusOK)
	})
}

func TestForwardedForIsIgnoredUnlessProxyIsTrusted(t *testing.T) {
	e := newEnv(t, func(o *api.Options) { o.AuthRatePerMin = 2 })

	// Sin proxy de confianza, cambiar X-Forwarded-For no debe servir para esquivar el límite.
	for i := 0; i < 2; i++ {
		loginAttempt(e, "a"+strconv.Itoa(i)+"@x.com", "x", withHeader("X-Forwarded-For", "203.0.113."+strconv.Itoa(i)))
	}
	res := loginAttempt(e, "a9@x.com", "x", withHeader("X-Forwarded-For", "203.0.113.99"))
	expectStatus(t, res, http.StatusTooManyRequests)
}

func TestTrustedProxyUsesTheClientIP(t *testing.T) {
	e := newEnv(t, func(o *api.Options) {
		o.AuthRatePerMin = 2
		o.TrustProxy = true
	})
	proxy := withRemoteAddr("10.0.0.5:5555") // siempre llega desde el proxy

	// Un cliente agota su límite; el proxy añade su IP real al final de la cabecera.
	// Lo que el cliente escriba antes (aquí 1.1.1.1, falso) se ignora.
	for i := 0; i < 2; i++ {
		res := loginAttempt(e, "a"+strconv.Itoa(i)+"@x.com", "x", proxy, withHeader("X-Forwarded-For", "1.1.1.1, 203.0.113.50"))
		expectStatus(t, res, http.StatusUnauthorized)
	}
	res := loginAttempt(e, "a9@x.com", "x", proxy, withHeader("X-Forwarded-For", "2.2.2.2, 203.0.113.50"))
	expectStatus(t, res, http.StatusTooManyRequests)

	// Otro cliente detrás del mismo proxy no se ve afectado.
	other := loginAttempt(e, "b@x.com", "x", proxy, withHeader("X-Forwarded-For", "203.0.113.51"))
	expectStatus(t, other, http.StatusUnauthorized)
}

func TestLoginLockoutByEmail(t *testing.T) {
	e := newEnv(t, func(o *api.Options) {
		o.LoginMaxFailures = 3
		o.LoginLockout = 15 * time.Minute
	})
	ana := e.customer("ana")

	for i := 1; i <= 3; i++ {
		expectStatus(t, loginAttempt(e, ana.Email, "incorrecta"), http.StatusUnauthorized)
	}

	t.Run("bloqueada, ni la contraseña correcta entra", func(t *testing.T) {
		res := loginAttempt(e, ana.Email, testPassword)
		expectStatus(t, res, http.StatusTooManyRequests)
		retry, _ := strconv.Atoi(res.Header.Get("Retry-After"))
		if retry < 14*60 || retry > 15*60 {
			t.Errorf("Retry-After = %d s, se esperaban ~15 min", retry)
		}
	})

	t.Run("el bloqueo ignora mayúsculas y espacios", func(t *testing.T) {
		expectStatus(t, loginAttempt(e, "  ANA@cliente.test ", testPassword), http.StatusTooManyRequests)
	})

	t.Run("otros emails no se ven afectados", func(t *testing.T) {
		luis := e.agent("luis")
		expectStatus(t, loginAttempt(e, luis.Email, testPassword), http.StatusOK)
	})
}

func TestLoginLockoutAlsoAppliesToUnknownEmails(t *testing.T) {
	// Si solo se bloquearan las cuentas que existen, el bloqueo revelaría cuáles existen.
	e := newEnv(t, func(o *api.Options) { o.LoginMaxFailures = 3 })
	for i := 0; i < 3; i++ {
		expectStatus(t, loginAttempt(e, "nadie@x.com", "x"), http.StatusUnauthorized)
	}
	expectStatus(t, loginAttempt(e, "nadie@x.com", "x"), http.StatusTooManyRequests)
}

func TestSuccessfulLoginResetsFailures(t *testing.T) {
	e := newEnv(t, func(o *api.Options) { o.LoginMaxFailures = 3 })
	ana := e.customer("ana")

	for round := 0; round < 3; round++ {
		loginAttempt(e, ana.Email, "mal")
		loginAttempt(e, ana.Email, "mal")
		expectStatus(t, loginAttempt(e, ana.Email, testPassword), http.StatusOK) // reinicia el contador
	}
}
