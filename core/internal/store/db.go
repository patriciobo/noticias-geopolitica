// Package store maneja la conexión a la base de datos de suscriptores
// (Postgres, hoy Neon free tier) y su schema. Es plomería genérica de
// conexión — la lógica de dominio de "suscriptor" vive en internal/subscriber.
package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open abre un pool de conexión a Postgres, lo verifica con un ping y
// asegura que el schema esperado exista. maxOpenConns bajo por default:
// el free tier de Neon limita conexiones concurrentes.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: no se pudo abrir la conexión: %w", err)
	}
	db.SetMaxOpenConns(5)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: no se pudo conectar a la base: %w", err)
	}

	if err := EnsureSchema(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: no se pudo asegurar el schema: %w", err)
	}

	return db, nil
}
