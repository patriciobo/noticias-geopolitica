// Package subscriber contiene el dominio y el repositorio de suscriptores
// del newsletter — separado de internal/store, que es solo la plomería de
// conexión/schema.
package subscriber

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// Subscriber es la información mínima que guardamos: un email y su estado
// de alta/confirmación/baja. Nada de nombre ni otros datos personales.
//
// Doble opt-in: una alta nueva queda pendiente (ConfirmedAt nil) hasta que
// la persona hace click en el mail de confirmación. Solo los confirmados y
// no dados de baja reciben el newsletter — así nadie puede anotar emails
// ajenos y convertir el newsletter en spam (o quemar la cuota de Brevo).
type Subscriber struct {
	ID               int64
	Email            string
	UnsubscribeToken string
	ConfirmToken     *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	UnsubscribedAt   *time.Time
	ConfirmedAt      *time.Time
	ConfirmSentAt    *time.Time
}

// Active dice si esta fila recibe el newsletter.
func (s *Subscriber) Active() bool {
	return s.ConfirmedAt != nil && s.UnsubscribedAt == nil
}

var ErrInvalidEmail = errors.New("subscriber: email inválido")

// ConfirmTokenTTL es cuánto vale un link de confirmación. Pasado eso la
// fila pendiente se purga (PurgeStalePending) y hay que volver a anotarse.
const ConfirmTokenTTL = 7 * 24 * time.Hour

// Outcome describe qué pasó con un pedido de alta, para que el caller sepa
// si tiene que mandar el mail de confirmación. La respuesta HTTP es la
// misma en todos los casos (ver cmd/api) — esto es solo para uso interno.
type Outcome int

const (
	// OutcomeNeedsConfirmation: alta nueva, reactivación de alguien dado
	// de baja, o pendiente cuyo último mail ya es viejo. Hay que mandar el
	// mail de confirmación con s.ConfirmToken.
	OutcomeNeedsConfirmation Outcome = iota
	// OutcomeAlreadyActive: ya confirmado y activo, no se hace nada.
	OutcomeAlreadyActive
	// OutcomeThrottled: pendiente con un mail de confirmación enviado hace
	// poco — no se reenvía, para que el form no sirva para bombardear una
	// casilla ajena con mails de confirmación.
	OutcomeThrottled
)

// NormalizeEmail valida y normaliza (trim + lower) un email ingresado por
// el usuario. net/mail.ParseAddress alcanza de sobra acá, sin traer una
// librería de validación de terceros.
func NormalizeEmail(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if len(trimmed) > 254 { // largo máximo de una dirección según RFC 5321
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed {
		return "", ErrInvalidEmail
	}
	return trimmed, nil
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("subscriber: no se pudo generar el token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const selectColumns = `id, email, unsubscribe_token, confirm_token, created_at, updated_at, unsubscribed_at, confirmed_at, confirm_sent_at`

type scanner interface {
	Scan(dest ...any) error
}

func scanSubscriber(row scanner) (*Subscriber, error) {
	var s Subscriber
	if err := row.Scan(&s.ID, &s.Email, &s.UnsubscribeToken, &s.ConfirmToken, &s.CreatedAt, &s.UpdatedAt, &s.UnsubscribedAt, &s.ConfirmedAt, &s.ConfirmSentAt); err != nil {
		return nil, err
	}
	return &s, nil
}

// Subscribe registra un pedido de alta. No activa a nadie: deja la fila
// pendiente con un confirm_token nuevo y devuelve OutcomeNeedsConfirmation
// para que el caller mande el mail. Si ya estaba activo, o pendiente con un
// mail enviado hace menos de resendAfter, no toca nada.
//
// Alguien dado de baja que se vuelve a anotar tiene que confirmar de nuevo:
// si no, cualquiera podría re-suscribir a una persona que se dio de baja.
func (r *Repository) Subscribe(ctx context.Context, email string, resendAfter time.Duration) (*Subscriber, Outcome, error) {
	email, err := NormalizeEmail(email)
	if err != nil {
		return nil, 0, err
	}
	unsubToken, err := generateToken()
	if err != nil {
		return nil, 0, err
	}
	confirmToken, err := generateToken()
	if err != nil {
		return nil, 0, err
	}

	q := `
INSERT INTO subscribers (email, unsubscribe_token, confirmed_at, confirm_token, confirm_sent_at)
VALUES ($1, $2, NULL, $3, now())
ON CONFLICT (email) DO UPDATE SET
	confirmed_at    = NULL,
	unsubscribed_at = NULL,
	confirm_token   = EXCLUDED.confirm_token,
	confirm_sent_at = now(),
	updated_at      = now()
	WHERE (subscribers.confirmed_at IS NULL OR subscribers.unsubscribed_at IS NOT NULL)
	  AND (subscribers.confirm_sent_at IS NULL OR subscribers.confirm_sent_at < now() - $4 * interval '1 second')
RETURNING ` + selectColumns
	row := r.db.QueryRowContext(ctx, q, email, unsubToken, confirmToken, int64(resendAfter.Seconds()))
	s, err := scanSubscriber(row)
	if err == nil {
		return s, OutcomeNeedsConfirmation, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, 0, fmt.Errorf("subscriber: no se pudo suscribir: %w", err)
	}

	// El ON CONFLICT no actualizó nada: ya activo, o pendiente con un mail
	// reciente. Traer la fila para distinguir los dos casos.
	s, err = r.getByEmail(ctx, email)
	if err != nil {
		return nil, 0, err
	}
	if s.Active() {
		return s, OutcomeAlreadyActive, nil
	}
	return s, OutcomeThrottled, nil
}

func (r *Repository) getByEmail(ctx context.Context, email string) (*Subscriber, error) {
	q := `SELECT ` + selectColumns + ` FROM subscribers WHERE email = $1`
	s, err := scanSubscriber(r.db.QueryRowContext(ctx, q, email))
	if err != nil {
		return nil, fmt.Errorf("subscriber: no se pudo leer la fila existente: %w", err)
	}
	return s, nil
}

// Confirm activa la suscripción asociada a un confirm_token vigente.
// Devuelve false si el token no existe, ya se usó o venció — el caller
// muestra el mismo mensaje genérico en todos esos casos.
func (r *Repository) Confirm(ctx context.Context, token string) (bool, error) {
	const q = `
UPDATE subscribers
SET confirmed_at = now(), confirm_token = NULL, unsubscribed_at = NULL, updated_at = now()
WHERE confirm_token = $1
  AND confirmed_at IS NULL
  AND confirm_sent_at > now() - $2 * interval '1 second'`
	res, err := r.db.ExecContext(ctx, q, token, int64(ConfirmTokenTTL.Seconds()))
	if err != nil {
		return false, fmt.Errorf("subscriber: no se pudo confirmar: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Unsubscribe da de baja por token. Idempotente a propósito: un token
// inválido o ya dado de baja se trata igual que un éxito, para no dejar
// que el endpoint sirva para verificar validez de tokens ajenos.
func (r *Repository) Unsubscribe(ctx context.Context, token string) error {
	const q = `UPDATE subscribers SET unsubscribed_at = now(), updated_at = now() WHERE unsubscribe_token = $1 AND unsubscribed_at IS NULL`
	if _, err := r.db.ExecContext(ctx, q, token); err != nil {
		return fmt.Errorf("subscriber: no se pudo dar de baja: %w", err)
	}
	return nil
}

// ConfirmationsSentSince cuenta cuántas filas recibieron un mail de
// confirmación en la ventana dada. Es el tope global que protege la cuota
// diaria de Brevo aunque alguien reparta el abuso entre muchas IPs.
// Cuenta filas, no envíos (un reenvío pisa confirm_sent_at), así que es una
// cota inferior — suficiente porque los reenvíos por fila ya están
// limitados por resendAfter.
func (r *Repository) ConfirmationsSentSince(ctx context.Context, window time.Duration) (int, error) {
	const q = `SELECT count(*) FROM subscribers WHERE confirm_sent_at > now() - $1 * interval '1 second'`
	var n int
	if err := r.db.QueryRowContext(ctx, q, int64(window.Seconds())).Scan(&n); err != nil {
		return 0, fmt.Errorf("subscriber: no se pudo contar confirmaciones: %w", err)
	}
	return n, nil
}

// PurgeStalePending borra altas que nunca se confirmaron dentro de
// ConfirmTokenTTL — no guardamos emails de gente que no pidió (o no
// terminó de pedir) el newsletter. Filas que alguna vez estuvieron
// confirmadas no se tocan.
func (r *Repository) PurgeStalePending(ctx context.Context) (int64, error) {
	const q = `
DELETE FROM subscribers
WHERE confirmed_at IS NULL
  AND confirm_sent_at < now() - $1 * interval '1 second'`
	res, err := r.db.ExecContext(ctx, q, int64(ConfirmTokenTTL.Seconds()))
	if err != nil {
		return 0, fmt.Errorf("subscriber: no se pudo purgar pendientes: %w", err)
	}
	return res.RowsAffected()
}

// ListActive trae los suscriptores que deben recibir el newsletter hoy:
// confirmados y no dados de baja.
func (r *Repository) ListActive(ctx context.Context) ([]Subscriber, error) {
	q := `SELECT ` + selectColumns + ` FROM subscribers WHERE unsubscribed_at IS NULL AND confirmed_at IS NOT NULL ORDER BY id`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("subscriber: no se pudo listar activos: %w", err)
	}
	defer rows.Close()

	var out []Subscriber
	for rows.Next() {
		s, err := scanSubscriber(rows)
		if err != nil {
			return nil, fmt.Errorf("subscriber: no se pudo leer fila: %w", err)
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}
