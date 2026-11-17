//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/organization"
	"neo360-api/internal/testutil"
)

func TestOrganizationRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewOrganizationRepository(database)
	ctx := context.Background()

	devs, err := repo.List(ctx, organization.KindDeveloper)
	require.NoError(t, err)
	require.Len(t, devs, 1, "a incorporadora do Boa Vista vem no seed")
	assert.Equal(t, "Dwell", devs[0].Name)
	assert.Equal(t, "/logos/incorporadora-dwell.jpg", devs[0].LogoURL)
	assert.Equal(t, "Gramado", devs[0].City)
	seeded, err := repo.List(ctx, organization.KindAgency)
	require.NoError(t, err)
	require.Len(t, seeded, 21, "Omni e as 20 imobiliárias da Serra Gaúcha e de Porto Alegre vêm pré-cadastradas")
	assert.Equal(t, "Omni", seeded[0].Name, "sem cidade vem primeiro")
	assert.Equal(t, "/logos/imobiliaria-omni.png", seeded[0].LogoURL)
	cities := map[string]int{}
	for _, a := range seeded {
		cities[a.City]++
		assert.True(t, a.Active, a.Name)
	}
	assert.Equal(t, map[string]int{"": 1, "Gramado": 3, "Canela": 7, "Caxias do Sul": 2, "Bento Gonçalves": 4, "Porto Alegre": 4}, cities)

	o, err := repo.Create(ctx, organization.Request{Kind: organization.KindAgency, Name: "Serra Imóveis", City: "Nova Petrópolis", LogoURL: "/logos/serra.png", Active: true})
	require.NoError(t, err)
	assert.Equal(t, "/logos/serra.png", o.LogoURL)
	assert.Equal(t, "Nova Petrópolis", o.City)

	all, err := repo.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, all, 23)
	agencies, err := repo.List(ctx, organization.KindAgency)
	require.NoError(t, err)
	assert.Len(t, agencies, 22)

	upd, err := repo.Update(ctx, o.ID, organization.Request{Kind: organization.KindAgency, Name: "Omni Imóveis", Email: "a@omni.com"})
	require.NoError(t, err)
	assert.Equal(t, "Omni Imóveis", upd.Name)
	assert.False(t, upd.Active)

	got, err := repo.Get(ctx, o.ID)
	require.NoError(t, err)
	assert.Equal(t, "a@omni.com", got.Email)

	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "nao-e-uuid"} {
		_, err = repo.Get(ctx, id)
		assert.ErrorIs(t, err, organization.ErrNotFound)
		_, err = repo.Update(ctx, id, organization.Request{Kind: organization.KindAgency, Name: "x"})
		assert.ErrorIs(t, err, organization.ErrNotFound)
	}

	require.NoError(t, database.Close())
	_, err = repo.List(ctx, "")
	assert.Error(t, err)
	_, err = repo.Create(ctx, organization.Request{Kind: organization.KindAgency, Name: "x"})
	assert.Error(t, err)
	_, err = repo.Get(ctx, o.ID)
	assert.Error(t, err)
}
