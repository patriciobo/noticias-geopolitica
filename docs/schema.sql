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
	unsubscribed_at   TIMESTAMPTZ NULL, -- NULL = no dado de baja
	confirmed_at      TIMESTAMPTZ NULL, -- NULL = pendiente de confirmar (doble opt-in)
	confirm_token     TEXT NULL,        -- token del mail de confirmación; NULL una vez confirmado
	confirm_sent_at   TIMESTAMPTZ NULL  -- último envío del mail de confirmación (throttle + tope diario)
);

-- Activo (recibe el newsletter) = confirmed_at IS NOT NULL AND unsubscribed_at IS NULL.

CREATE UNIQUE INDEX IF NOT EXISTS subscribers_email_idx ON subscribers (email);
CREATE UNIQUE INDEX IF NOT EXISTS subscribers_unsub_token_idx ON subscribers (unsubscribe_token);
CREATE INDEX IF NOT EXISTS subscribers_active_idx ON subscribers (unsubscribed_at) WHERE unsubscribed_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS subscribers_confirm_token_idx ON subscribers (confirm_token) WHERE confirm_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS subscribers_confirm_sent_idx ON subscribers (confirm_sent_at) WHERE confirm_sent_at IS NOT NULL;
