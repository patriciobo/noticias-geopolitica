package store

import (
	"context"
	"database/sql"
	"fmt"
)

// EnsureSchema crea las tablas/índices necesarios si no existen. Es
// idempotente a propósito: no hay framework de migraciones (goose,
// golang-migrate) porque el schema hoy es una sola tabla — correr esto en
// cada arranque de cmd/api y cmd/newsletter alcanza y evita un paso manual
// de "correr un script SQL" que Render no ofrece de forma cómoda.
//
// docs/schema.sql documenta este mismo DDL para referencia humana rápida
// (ej. al debuggear con psql), pero la fuente de verdad es este archivo.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS subscribers (
	id                BIGSERIAL PRIMARY KEY,
	email             TEXT NOT NULL,
	unsubscribe_token TEXT NOT NULL,
	created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
	unsubscribed_at   TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS subscribers_email_idx ON subscribers (email);
CREATE UNIQUE INDEX IF NOT EXISTS subscribers_unsub_token_idx ON subscribers (unsubscribe_token);
CREATE INDEX IF NOT EXISTS subscribers_active_idx ON subscribers (unsubscribed_at) WHERE unsubscribed_at IS NULL;
`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("no se pudo aplicar el schema de subscribers: %w", err)
	}
	return nil
}
