//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/account"
	"neo360-api/internal/testutil"
)

func TestAccountRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewAccountRepository(database)
	ctx := context.Background()

	id, err := repo.CreateAdmin(ctx, "Central", "Central@Neo360.test", "hash")
	require.NoError(t, err)
	_, err = repo.CreateAdmin(ctx, "Outra", "central@neo360.test", "hash")
	assert.ErrorIs(t, err, account.ErrEmailTaken)

	c, err := repo.Credentials(ctx, "central@neo360.test")
	require.NoError(t, err)
	assert.Equal(t, account.Credentials{UserID: id, Role: "admin", PasswordHash: "hash", Active: true}, c)
	_, err = repo.Credentials(ctx, "ninguem@neo360.test")
	assert.ErrorIs(t, err, account.ErrNotFound)

	u, err := repo.User(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "central@neo360.test", u.Email, "e-mail gravado em minúsculas")
	assert.False(t, u.MustChangePassword)

	// ADR-0020: reset grava o hash da provisória e marca a troca; a troca limpa a marca.
	require.NoError(t, repo.ResetPassword(ctx, id, "hash-provisoria"))
	h, err := repo.PasswordHash(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "hash-provisoria", h)
	list, err := repo.ListAccounts(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, list)
	var me account.Account
	for _, a := range list {
		if a.ID == id {
			me = a
		}
	}
	assert.True(t, me.MustChangePassword)
	assert.NotNil(t, me.PasswordResetAt)
	assert.Equal(t, "admin", me.Role)
	u, err = repo.User(ctx, id)
	require.NoError(t, err)
	assert.True(t, u.MustChangePassword)
	require.NoError(t, repo.ChangePassword(ctx, id, "hash-nova"))
	u, err = repo.User(ctx, id)
	require.NoError(t, err)
	assert.False(t, u.MustChangePassword)
	assert.ErrorIs(t, repo.ResetPassword(ctx, "00000000-0000-0000-0000-000000000000", "h"), account.ErrNotFound)
	assert.ErrorIs(t, repo.ResetPassword(ctx, "nao-e-uuid", "h"), account.ErrNotFound)
	_, err = repo.PasswordHash(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, account.ErrNotFound)

	_, err = database.ExecContext(ctx, `UPDATE users SET active = false WHERE id = $1`, id)
	require.NoError(t, err)
	_, err = repo.User(ctx, id)
	assert.ErrorIs(t, err, account.ErrNotFound, "conta desativada some")
	_, err = repo.User(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, account.ErrNotFound)

	require.NoError(t, database.Close())
	_, err = repo.User(ctx, id)
	assert.Error(t, err)
	_, err = repo.CreateAdmin(ctx, "x", "x@x.com", "h")
	assert.Error(t, err)
	_, err = repo.ListAccounts(ctx)
	assert.Error(t, err)
	assert.Error(t, repo.ChangePassword(ctx, id, "h"))
}
