// Package testutil ayuda a escribir pruebas contra un PostgreSQL real.
package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MADGRISMAD/MiColmena/backend/internal/db"
)

// EnvVar es la variable con la URL de una base de datos donde las pruebas pueden crear esquemas.
const EnvVar = "TEST_DATABASE_URL"

// NewPool devuelve un pool conectado a un esquema nuevo y con las migraciones aplicadas.
// El esquema se borra al terminar la prueba. Sin TEST_DATABASE_URL, la prueba se salta.
func NewPool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	pool := NewEmptyPool(t)
	if err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrar esquema de pruebas: %v", err)
	}
	return pool
}

// NewEmptyPool es como NewPool pero sin aplicar las migraciones.
func NewEmptyPool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv(EnvVar)
	if url == "" {
		t.Skipf("define %s para ejecutar las pruebas con PostgreSQL", EnvVar)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("conectar a %s: %v", EnvVar, err)
	}

	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	schema := "test_" + hex.EncodeToString(suffix[:])
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		admin.Close()
		t.Fatalf("crear esquema: %v", err)
	}

	// Con search_path fijo, todas las tablas sin prefijo caen en este esquema.
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		admin.Close()
		t.Fatalf("abrir pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Errorf("borrar esquema %s: %v", schema, err)
		}
		admin.Close()
	})
	return pool
}
