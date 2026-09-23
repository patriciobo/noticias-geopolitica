package subscriber_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"noticias/core/internal/store"
	"noticias/core/internal/subscriber"
)

// TestRepositoryAgainstPostgres corre el flujo completo de doble opt-in
// contra un Postgres real. Se saltea si no hay TEST_DATABASE_URL — NO
// apuntar a la base de producción: el test borra la tabla subscribers.
//
//	TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable go test ./internal/subscriber/
func TestRepositoryAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL no seteada")
	}
	ctx := context.Background()

	// Arranca desde el schema previo al doble opt-in, con una fila ya
	// suscripta, para probar que la migración la deja activa.
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	_, err = raw.Exec(`
DROP TABLE IF EXISTS subscribers;
CREATE TABLE subscribers (
	id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL, unsubscribe_token TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	unsubscribed_at TIMESTAMPTZ NULL);
CREATE UNIQUE INDEX subscribers_email_idx ON subscribers (email);
INSERT INTO subscribers (email, unsubscribe_token) VALUES ('viejo@ejemplo.com', 'tok-viejo');`)
	if err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := subscriber.NewRepository(db)

	activeEmails := func() []string {
		t.Helper()
		active, err := repo.ListActive(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, s := range active {
			out = append(out, s.Email)
		}
		return out
	}

	if got := activeEmails(); len(got) != 1 || got[0] != "viejo@ejemplo.com" {
		t.Fatalf("la fila previa a la migración debería quedar activa, got %v", got)
	}

	s, out, err := repo.Subscribe(ctx, "Nuevo@Ejemplo.com", time.Hour)
	if err != nil || out != subscriber.OutcomeNeedsConfirmation || s.ConfirmToken == nil {
		t.Fatalf("alta nueva: outcome=%v err=%v sub=%+v", out, err, s)
	}
	token := *s.ConfirmToken

	// Re-aplicar el schema (pasa en cada arranque) no debe confirmar pendientes.
	if err := store.EnsureSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if got := activeEmails(); len(got) != 1 {
		t.Fatalf("un pendiente no debe recibir el newsletter, activos=%v", got)
	}

	if _, out, _ := repo.Subscribe(ctx, "nuevo@ejemplo.com", time.Hour); out != subscriber.OutcomeThrottled {
		t.Fatalf("reintento inmediato: got %v, want OutcomeThrottled", out)
	}
	if n, _ := repo.ConfirmationsSentSince(ctx, 24*time.Hour); n != 1 {
		t.Fatalf("ConfirmationsSentSince = %d, want 1", n)
	}

	if ok, err := repo.Confirm(ctx, "token-inventado"); ok || err != nil {
		t.Fatalf("token inventado: ok=%v err=%v", ok, err)
	}
	if ok, err := repo.Confirm(ctx, token); !ok || err != nil {
		t.Fatalf("confirmar: ok=%v err=%v", ok, err)
	}
	if ok, _ := repo.Confirm(ctx, token); ok {
		t.Fatal("un token ya usado no debería volver a confirmar")
	}
	if got := activeEmails(); len(got) != 2 {
		t.Fatalf("confirmado debería estar activo, activos=%v", got)
	}
	if _, out, _ := repo.Subscribe(ctx, "nuevo@ejemplo.com", time.Hour); out != subscriber.OutcomeAlreadyActive {
		t.Fatalf("ya activo: got %v", out)
	}

	// Baja y re-alta: hay que confirmar de nuevo.
	active, _ := repo.ListActive(ctx)
	if err := repo.Unsubscribe(ctx, active[1].UnsubscribeToken); err != nil {
		t.Fatal(err)
	}
	s, out, err = repo.Subscribe(ctx, "nuevo@ejemplo.com", 0)
	if err != nil || out != subscriber.OutcomeNeedsConfirmation || s.Active() {
		t.Fatalf("re-alta tras baja: outcome=%v active=%v err=%v", out, s.Active(), err)
	}

	// Un link vencido no confirma, y el pendiente vencido se purga.
	if _, err := db.Exec(`UPDATE subscribers SET confirm_sent_at = now() - interval '8 days' WHERE email = 'nuevo@ejemplo.com'`); err != nil {
		t.Fatal(err)
	}
	if ok, _ := repo.Confirm(ctx, *s.ConfirmToken); ok {
		t.Fatal("un token vencido no debería confirmar")
	}
	if n, err := repo.PurgeStalePending(ctx); err != nil || n != 1 {
		t.Fatalf("PurgeStalePending = %d, %v; want 1", n, err)
	}
	if got := activeEmails(); len(got) != 1 || got[0] != "viejo@ejemplo.com" {
		t.Fatalf("solo debería quedar el suscriptor previo, activos=%v", got)
	}
}
