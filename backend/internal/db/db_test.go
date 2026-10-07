package db_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/MADGRISMAD/MiColmena/backend/internal/db"
	"github.com/MADGRISMAD/MiColmena/backend/internal/testutil"
)

func TestMigrateCreatesSchemaAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	pool := testutil.NewEmptyPool(t)

	for i := 0; i < 2; i++ {
		if err := db.Migrate(ctx, pool); err != nil {
			t.Fatalf("migrar (vez %d): %v", i+1, err)
		}
	}

	var applied int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob("migrations/*.sql")
	if applied != len(files) {
		t.Errorf("migraciones registradas = %d, se esperaban %d (no deben repetirse)", applied, len(files))
	}

	for _, table := range []string{"users", "tickets", "ticket_comments"} {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("falta la tabla %s", table)
		}
	}
}

func TestMigrateConcurrentInstances(t *testing.T) {
	ctx := context.Background()
	pool := testutil.NewEmptyPool(t)

	// Dos instancias de la API arrancando a la vez no deben pisarse.
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- db.Migrate(ctx, pool)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("migración concurrente: %v", err)
		}
	}
}

func TestConstraints(t *testing.T) {
	ctx := context.Background()
	pool := testutil.NewPool(t)

	insert := func(email, role string) error {
		_, err := pool.Exec(ctx,
			`INSERT INTO users (name, email, password_hash, role, org_id) VALUES ('x', $1, 'h', $2, 1)`, email, role)
		return err
	}

	if err := insert("Ana@X.com", "customer"); err != nil {
		t.Fatal(err)
	}
	if err := insert("ana@x.com", "customer"); err == nil {
		t.Error("el email debería ser único en la empresa sin distinguir mayúsculas")
	}
	if err := insert("b@x.com", "superuser"); err == nil {
		t.Error("un rol que no existe debería rechazarse")
	}
}
