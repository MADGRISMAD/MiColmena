package auth

import (
	"testing"
	"time"
)

func TestPassword(t *testing.T) {
	hash, err := HashPassword("secreto123")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "secreto123") {
		t.Error("la contraseña correcta no se aceptó")
	}
	if CheckPassword(hash, "otra-cosa") {
		t.Error("se aceptó una contraseña incorrecta")
	}
}

func TestTokenRoundTrip(t *testing.T) {
	issuer := NewIssuer([]byte("0123456789abcdef0123456789abcdef"), time.Hour)
	token, _, err := issuer.Issue(42, RoleAgent)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := issuer.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID() != 42 || claims.Role != RoleAgent || !claims.IsStaff() {
		t.Errorf("claims inesperados: %+v", claims)
	}
}

func TestTokenRejected(t *testing.T) {
	issuer := NewIssuer([]byte("0123456789abcdef0123456789abcdef"), time.Hour)
	other := NewIssuer([]byte("ffffffffffffffffffffffffffffffff"), time.Hour)
	expired := NewIssuer([]byte("0123456789abcdef0123456789abcdef"), -time.Minute)

	foreign, _, _ := other.Issue(1, RoleAdmin)
	old, _, _ := expired.Issue(1, RoleAdmin)

	for name, token := range map[string]string{
		"otra clave": foreign,
		"expirado":   old,
		"basura":     "no-es-un-token",
	} {
		if _, err := issuer.Parse(token); err == nil {
			t.Errorf("%s: se aceptó el token", name)
		}
	}
}
