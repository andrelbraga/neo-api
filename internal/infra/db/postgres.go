package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// ErrInvalidSchema protege o nome do schema, que entra no SQL sem parâmetro.
var ErrInvalidSchema = errors.New("DB_SCHEMA inválido: use letras minúsculas, números e _")

var schemaName = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// Open abre o pool de conexões e confirma que o banco responde. Com schema, todas as conexões enxergam só
// esse schema (search_path) e ele é criado se não existir: o Neo360 divide o banco com outro projeto sem
// tocar nas tabelas dele (ADR-0018). Sem schema, vale o padrão do banco (public).
func Open(ctx context.Context, dsn, schema string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if schema != "" {
		if !schemaName.MatchString(schema) {
			return nil, ErrInvalidSchema
		}
		cfg.RuntimeParams["search_path"] = schema
	}
	database := stdlib.OpenDB(*cfg)

	database.SetMaxOpenConns(10)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := database.PingContext(pingCtx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if schema != "" {
		if _, err := database.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS "`+schema+`"`); err != nil {
			_ = database.Close()
			return nil, fmt.Errorf("create schema: %w", err)
		}
	}
	return database, nil
}
