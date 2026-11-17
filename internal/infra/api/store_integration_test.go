//go:build integration

package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/pkg/middleware"
	"neo360-api/internal/testutil"
)

func TestStore_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../sql/migrations")
	s := store{db: database}
	ctx := context.Background()

	var id string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Corretor', 'c@neo360.test', 'h', 'corretor') RETURNING id`).Scan(&id))

	role, err := s.ActiveRole(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "corretor", role)

	_, err = s.ActiveRole(ctx, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, middleware.ErrAccountNotFound)
	_, err = s.ActiveRole(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, middleware.ErrAccountNotFound)

	_, err = database.ExecContext(ctx, `UPDATE users SET active = false WHERE id = $1`, id)
	require.NoError(t, err)
	_, err = s.ActiveRole(ctx, id)
	assert.ErrorIs(t, err, middleware.ErrAccountNotFound, "desativado perde o acesso na hora (RN-01.5)")

	require.NoError(t, database.Close())
	_, err = s.ActiveRole(ctx, id)
	assert.Error(t, err)
}
