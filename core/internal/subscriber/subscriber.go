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
// de alta/baja. Nada de nombre ni otros datos personales.
type Subscriber struct {
	ID               int64
	Email            string
	UnsubscribeToken string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	UnsubscribedAt   *time.Time
}

var ErrInvalidEmail = errors.New("subscriber: email inválido")

// NormalizeEmail valida y normaliza (trim + lower) un email ingresado por
// el usuario. net/mail.ParseAddress alcanza de sobra acá, sin traer una
// librería de validación de terceros.
func NormalizeEmail(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
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

// Subscribe da de alta un email, o lo reactiva si estaba dado de baja. Si ya
// estaba activo, es un no-op que igual devuelve la fila (mismo token de
// siempre — no se rota en reactivaciones, no hace falta para el modelo de
// amenaza acá).
func (r *Repository) Subscribe(ctx context.Context, email string) (*Subscriber, error) {
	email, err := NormalizeEmail(email)
	if err != nil {
		return nil, err
	}
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	const q = `
INSERT INTO subscribers (email, unsubscribe_token)
VALUES ($1, $2)
ON CONFLICT (email) DO UPDATE SET
	unsubscribed_at = NULL,
	updated_at = now()
	WHERE subscribers.unsubscribed_at IS NOT NULL
RETURNING id, email, unsubscribe_token, created_at, updated_at, unsubscribed_at
`
	row := r.db.QueryRowContext(ctx, q, email, token)
	var s Subscriber
	if err := row.Scan(&s.ID, &s.Email, &s.UnsubscribeToken, &s.CreatedAt, &s.UpdatedAt, &s.UnsubscribedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// El ON CONFLICT no actualizó nada (ya estaba activo) — traer la
			// fila existente tal cual está.
			return r.getByEmail(ctx, email)
		}
		return nil, fmt.Errorf("subscriber: no se pudo suscribir: %w", err)
	}
	return &s, nil
}

func (r *Repository) getByEmail(ctx context.Context, email string) (*Subscriber, error) {
	const q = `SELECT id, email, unsubscribe_token, created_at, updated_at, unsubscribed_at FROM subscribers WHERE email = $1`
	row := r.db.QueryRowContext(ctx, q, email)
	var s Subscriber
	if err := row.Scan(&s.ID, &s.Email, &s.UnsubscribeToken, &s.CreatedAt, &s.UpdatedAt, &s.UnsubscribedAt); err != nil {
		return nil, fmt.Errorf("subscriber: no se pudo leer la fila existente: %w", err)
	}
	return &s, nil
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

// ListActive trae los suscriptores que deben recibir el newsletter hoy.
func (r *Repository) ListActive(ctx context.Context) ([]Subscriber, error) {
	const q = `SELECT id, email, unsubscribe_token, created_at, updated_at, unsubscribed_at FROM subscribers WHERE unsubscribed_at IS NULL ORDER BY id`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("subscriber: no se pudo listar activos: %w", err)
	}
	defer rows.Close()

	var out []Subscriber
	for rows.Next() {
		var s Subscriber
		if err := rows.Scan(&s.ID, &s.Email, &s.UnsubscribeToken, &s.CreatedAt, &s.UpdatedAt, &s.UnsubscribedAt); err != nil {
			return nil, fmt.Errorf("subscriber: no se pudo leer fila: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
