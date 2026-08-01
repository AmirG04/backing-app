package db

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(databaseURL string) (*pgxpool.Pool, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("error creando pool de postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("error haciendo ping a postgres: %w", err)
	}
	return pool, nil
}

// Migrate crea las tablas necesarias si no existen.
// Para una prueba tecnica esto es suficiente; en produccion se usaria
// una herramienta de migraciones versionadas (goose, migrate, atlas, etc).
func Migrate(pool *pgxpool.Pool) error {
	ctx := context.Background()
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		full_name TEXT NOT NULL,
		tigerbeetle_account_id TEXT UNIQUE NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		token_hash TEXT NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		revoked_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);
	`
	if _, err := pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pgcrypto;"); err != nil {
		log.Printf("aviso: no se pudo crear extension pgcrypto (puede que ya exista o falten permisos): %v", err)
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("error corriendo migracion: %w", err)
	}
	return nil
}
