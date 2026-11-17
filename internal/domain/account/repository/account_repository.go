package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"neo360-api/internal/domain/account"
	"neo360-api/internal/pkg/pgerr"
)

const (
	queryCredentials = `SELECT id, role, password_hash, active FROM users WHERE lower(email) = $1`
	queryUser        = `SELECT id, name, email, role FROM users WHERE id = $1 AND active`
	queryCreateAdmin = `INSERT INTO users (name, email, password_hash, role) VALUES ($1, lower($2), $3, 'admin') RETURNING id`
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Credentials(ctx context.Context, email string) (account.Credentials, error) {
	var c account.Credentials
	err := r.db.QueryRowContext(ctx, queryCredentials, email).Scan(&c.UserID, &c.Role, &c.PasswordHash, &c.Active)
	return c, notFound(err, "credentials")
}

func (r *AccountRepository) User(ctx context.Context, id string) (account.User, error) {
	var u account.User
	err := r.db.QueryRowContext(ctx, queryUser, id).Scan(&u.ID, &u.Name, &u.Email, &u.Role)
	return u, notFound(err, "user")
}

// CreateAdmin cria uma conta da equipe Neo360. Só o comando create-admin usa (não há cadastro aberto de admin).
func (r *AccountRepository) CreateAdmin(ctx context.Context, name, email, hash string) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, queryCreateAdmin, name, email, hash).Scan(&id)
	if pgerr.Unique(err, "users_email_key") {
		return "", account.ErrEmailTaken
	}
	if err != nil {
		return "", fmt.Errorf("create admin: %w", err)
	}
	return id, nil
}

func notFound(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) || pgerr.BadUUID(err) {
		return account.ErrNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
