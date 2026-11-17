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
}
