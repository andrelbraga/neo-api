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
	queryUser        = `SELECT id, name, email, role, must_change_password FROM users WHERE id = $1 AND active`
	queryAccounts    = `
		SELECT u.id, u.name, u.email, u.role, u.active, COALESCE(b.code, ''), u.must_change_password, u.password_reset_at, u.created_at
		FROM users u LEFT JOIN brokers b ON b.user_id = u.id ORDER BY u.role, lower(u.name)`
	queryPasswordHash  = `SELECT password_hash FROM users WHERE id = $1 AND active`
	queryResetPassword = `
		UPDATE users SET password_hash = $2, must_change_password = true, password_reset_at = now(), updated_at = now() WHERE id = $1`
	queryChangePassword = `UPDATE users SET password_hash = $2, must_change_password = false, updated_at = now() WHERE id = $1 AND active`
	queryCreateAdmin    = `INSERT INTO users (name, email, password_hash, role) VALUES ($1, lower($2), $3, 'admin') RETURNING id`
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
	err := r.db.QueryRowContext(ctx, queryUser, id).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.MustChangePassword)
	return u, notFound(err, "user")
}

// ListAccounts devolve todas as contas, admins primeiro, sem nenhum dado de senha.
func (r *AccountRepository) ListAccounts(ctx context.Context) ([]account.Account, error) {
	rows, err := r.db.QueryContext(ctx, queryAccounts)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	list := []account.Account{}
	for rows.Next() {
		var a account.Account
		var reset sql.NullTime
		if err := rows.Scan(&a.ID, &a.Name, &a.Email, &a.Role, &a.Active, &a.BrokerCode, &a.MustChangePassword, &reset, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		if reset.Valid {
			a.PasswordResetAt = &reset.Time
		}
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return list, nil
}

func (r *AccountRepository) PasswordHash(ctx context.Context, id string) (string, error) {
	var hash string
	err := r.db.QueryRowContext(ctx, queryPasswordHash, id).Scan(&hash)
	return hash, notFound(err, "password hash")
}

func (r *AccountRepository) ResetPassword(ctx context.Context, id, hash string) error {
	return r.update(ctx, queryResetPassword, id, hash, "reset password")
}

func (r *AccountRepository) ChangePassword(ctx context.Context, id, hash string) error {
	return r.update(ctx, queryChangePassword, id, hash, "change password")
}

func (r *AccountRepository) update(ctx context.Context, query, id, hash, op string) error {
	res, err := r.db.ExecContext(ctx, query, id, hash)
	if err != nil {
		return notFound(err, op)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return account.ErrNotFound
	}
	return nil
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
