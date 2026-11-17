package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neo360-api/internal/pkg/middleware"
)

const (
	invalidTextRepresentation = "22P02"

	// Corretor desativado pelo admin perde a sessão na hora (RN-01.5).
	queryActiveRole = `SELECT role FROM users WHERE id = $1 AND active`
	queryMustChange = `SELECT must_change_password FROM users WHERE id = $1`
)

// MustChangePassword diz se o usuário está com senha provisória (ADR-0020).
func (s store) MustChangePassword(ctx context.Context, userID string) (bool, error) {
	var must bool
	err := s.db.QueryRowContext(ctx, queryMustChange, userID).Scan(&must)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("must change password: %w", err)
	}
	return must, nil
}

// store é o que o middleware de autenticação precisa do banco: o papel atual de quem está ativo.
type store struct {
	db *sql.DB
}

func (s store) ActiveRole(ctx context.Context, userID string) (string, error) {
	var role string
	err := s.db.QueryRowContext(ctx, queryActiveRole, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", middleware.ErrAccountNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		return "", middleware.ErrAccountNotFound
	}
	if err != nil {
		return "", fmt.Errorf("active role: %w", err)
	}
	return role, nil
}
