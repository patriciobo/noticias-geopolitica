-- Documental únicamente. La fuente de verdad es
-- core/internal/store/schema.go (EnsureSchema), que aplica este mismo DDL
-- de forma idempotente en cada arranque de cmd/api y cmd/newsletter. Este
-- archivo no lo ejecuta nada — es solo para consultarlo rápido con psql.

CREATE TABLE IF NOT EXISTS subscribers (
	id                BIGSERIAL PRIMARY KEY,
	email             TEXT NOT NULL,
	unsubscribe_token TEXT NOT NULL,
	created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
	unsubscribed_at   TIMESTAMPTZ NULL -- NULL = activo
);

CREATE UNIQUE INDEX IF NOT EXISTS subscribers_email_idx ON subscribers (email);
CREATE UNIQUE INDEX IF NOT EXISTS subscribers_unsub_token_idx ON subscribers (unsubscribe_token);
CREATE INDEX IF NOT EXISTS subscribers_active_idx ON subscribers (unsubscribed_at) WHERE unsubscribed_at IS NULL;
