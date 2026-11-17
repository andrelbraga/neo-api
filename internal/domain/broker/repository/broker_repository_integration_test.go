//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/broker"
	"neo360-api/internal/testutil"
)

func TestBrokerRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBrokerRepository(database)
	ctx := context.Background()
	var agencyID, developerID string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT id FROM organizations WHERE kind = 'imobiliaria' AND name = 'Omni'`).Scan(&agencyID))
	require.NoError(t, database.QueryRowContext(ctx, `SELECT id FROM organizations WHERE kind = 'incorporadora'`).Scan(&developerID))

	req := broker.CreateRequest{Profile: broker.Profile{FirstName: "Fabiano", LastName: "Souza", Phone: "54999990000"},
		Email: "fab@exemplo.com", Password: "x"}
	b, err := repo.Create(ctx, req, "hash", "FAB-0001")
	require.NoError(t, err)
	assert.Equal(t, "FAB-0001", b.Code)
	assert.Equal(t, "fab@exemplo.com", b.Email)
	assert.True(t, b.Active)
	assert.Nil(t, b.AgencyID)
	assert.Zero(t, b.LeadsCount)

	var role string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT role FROM users WHERE id = $1`, b.UserID).Scan(&role))
	assert.Equal(t, "corretor", role)

	req.Email = "FAB@exemplo.com"
	_, err = repo.Create(ctx, req, "hash", "FAB-0002")
	assert.ErrorIs(t, err, broker.ErrEmailTaken, "e-mail não diferencia maiúsculas")
	req.Email = "outro@exemplo.com"
	_, err = repo.Create(ctx, req, "hash", "FAB-0001")
	assert.ErrorIs(t, err, broker.ErrCodeTaken)
	for _, bad := range []string{developerID, "nao-e-uuid"} {
		req.AgencyID = &bad
		_, err = repo.Create(ctx, req, "hash", "FAB-0003")
		assert.ErrorIs(t, err, broker.ErrAgencyNotFound, "incorporadora não é imobiliária")
	}
	req.AgencyID = &agencyID
	ana, err := repo.Create(ctx, broker.CreateRequest{Profile: broker.Profile{FirstName: "Ana", LastName: "Lima", Phone: "54988887777"},
		Email: "ana@exemplo.com", AgencyID: &agencyID}, "hash", "ANA-0001")
	require.NoError(t, err)
	assert.Equal(t, "Omni", ana.AgencyName)
	assert.Equal(t, "/logos/imobiliaria-omni.png", ana.AgencyLogo)

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "Ana", list[0].FirstName, "ordem alfabética")

	got, err := repo.GetByUser(ctx, b.UserID)
	require.NoError(t, err)
	assert.Equal(t, b.ID, got.ID)

	upd, err := repo.Update(ctx, b.ID, broker.UpdateRequest{Profile: broker.Profile{FirstName: "Fabiano", LastName: "Silva", Phone: "54911112222", Creci: "123"},
		AgencyID: &agencyID, Active: false})
	require.NoError(t, err)
	assert.Equal(t, "Silva", upd.LastName)
	assert.Equal(t, "Omni", upd.AgencyName)
	assert.False(t, upd.Active, "o admin desativa a conta")
	var name string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT name FROM users WHERE id = $1`, b.UserID).Scan(&name))
	assert.Equal(t, "Fabiano Silva", name, "o nome da conta acompanha o cadastro")

	prof, err := repo.UpdateProfile(ctx, b.UserID, broker.Profile{FirstName: "Fabi", LastName: "Silva", Phone: "54900000000"})
	require.NoError(t, err)
	assert.Equal(t, "Fabi", prof.FirstName)
	assert.False(t, prof.Active, "o perfil não reativa a conta")
	assert.Equal(t, "Omni", prof.AgencyName, "o perfil não mexe na imobiliária")

	missing := "00000000-0000-0000-0000-000000000000"
	for _, id := range []string{missing, "nao-e-uuid"} {
		_, err = repo.Get(ctx, id)
		assert.ErrorIs(t, err, broker.ErrNotFound)
		_, err = repo.GetByUser(ctx, id)
		assert.ErrorIs(t, err, broker.ErrNotFound)
		_, err = repo.Update(ctx, id, broker.UpdateRequest{})
		assert.ErrorIs(t, err, broker.ErrNotFound)
		_, err = repo.UpdateProfile(ctx, id, broker.Profile{})
		assert.ErrorIs(t, err, broker.ErrNotFound)
	}
	bad := "nao-e-uuid"
	_, err = repo.Update(ctx, b.ID, broker.UpdateRequest{AgencyID: &bad})
	assert.ErrorIs(t, err, broker.ErrAgencyNotFound)

	require.NoError(t, database.Close())
	_, err = repo.List(ctx)
	assert.Error(t, err)
	_, err = repo.Get(ctx, b.ID)
	assert.Error(t, err)
	_, err = repo.Create(ctx, req, "h", "X-1")
	assert.Error(t, err)
	_, err = repo.Update(ctx, b.ID, broker.UpdateRequest{})
	assert.Error(t, err)
	_, err = repo.UpdateProfile(ctx, b.UserID, broker.Profile{})
	assert.Error(t, err)
}
