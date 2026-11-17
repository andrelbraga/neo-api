//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/development"
	"neo360-api/internal/testutil"
)

func TestDevelopmentRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewDevelopmentRepository(database)
	ctx := context.Background()

	bv, err := repo.GetPublishedBySlug(ctx, "boa-vista-residencial")
	require.NoError(t, err, "o Boa Vista vem pré-carregado e publicado")
	assert.Equal(t, "Dwell", bv.DeveloperName)
	assert.Equal(t, "Gramado", bv.City)
	assert.NotEmpty(t, bv.Gallery)
	assert.NotEmpty(t, bv.Highlights)
	assert.Equal(t, "/showroom/boa-vista/maquete.html?embed=1", bv.TourURL)
	assert.Equal(t, "/showroom/boa-vista/index.html", bv.PresentationURL, "o Boa Vista usa a apresentação do protótipo")
	assert.Equal(t, development.LayoutA, bv.DefaultLayout, "o layout A é o padrão")

	var agencyID string
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO organizations (kind, name) VALUES ('imobiliaria', 'Omni') RETURNING id`).Scan(&agencyID))
	req := development.Request{Slug: "vale-verde", Name: "Vale Verde", Status: development.StatusBuilding, DeveloperID: bv.DeveloperID,
		Gallery: []development.Image{}, Highlights: []development.Highlight{{Label: "Área", Value: "70 m²"}}}
	d, err := repo.Create(ctx, req)
	require.NoError(t, err)
	assert.False(t, d.Published)
	assert.Equal(t, []development.Highlight{{Label: "Área", Value: "70 m²"}}, d.Highlights)

	_, err = repo.Create(ctx, req)
	assert.ErrorIs(t, err, development.ErrSlugTaken)
	for _, bad := range []string{agencyID, "nao-e-uuid"} {
		req.Slug, req.DeveloperID = "outro", &bad
		_, err = repo.Create(ctx, req)
		assert.ErrorIs(t, err, development.ErrDeveloperNotFound, "imobiliária não é incorporadora")
	}

	published, err := repo.List(ctx, true)
	require.NoError(t, err)
	assert.Len(t, published, 1, "o corretor só vê o publicado")
	all, err := repo.List(ctx, false)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	_, err = repo.GetPublishedBySlug(ctx, "vale-verde")
	assert.ErrorIs(t, err, development.ErrNotFound, "rascunho não abre no showroom")

	req.Slug, req.DeveloperID, req.Published, req.PresentationURL = "vale-verde", nil, true, "/showroom/vv/index.html"
	req.DefaultLayout = development.LayoutB
	upd, err := repo.Update(ctx, d.ID, req)
	require.NoError(t, err)
	assert.True(t, upd.Published)
	assert.Equal(t, "/showroom/vv/index.html", upd.PresentationURL)
	assert.Equal(t, development.LayoutB, upd.DefaultLayout)
	assert.Nil(t, upd.DeveloperID)

	req.Slug = "boa-vista-residencial"
	_, err = repo.Update(ctx, d.ID, req)
	assert.ErrorIs(t, err, development.ErrSlugTaken)
	bad := "nao-e-uuid"
	req.DeveloperID = &bad
	_, err = repo.Update(ctx, d.ID, req)
	assert.ErrorIs(t, err, development.ErrDeveloperNotFound)

	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "nao-e-uuid"} {
		_, err = repo.Get(ctx, id)
		assert.ErrorIs(t, err, development.ErrNotFound)
		_, err = repo.Update(ctx, id, development.Request{Slug: "z", Name: "z", Status: development.StatusReady})
		assert.ErrorIs(t, err, development.ErrNotFound)
	}

	require.NoError(t, database.Close())
	_, err = repo.List(ctx, false)
	assert.Error(t, err)
	_, err = repo.Create(ctx, development.Request{Slug: "w", Name: "w"})
	assert.Error(t, err)
	id := d.DeveloperID
	_, err = repo.Create(ctx, development.Request{Slug: "w", Name: "w", DeveloperID: id})
	assert.Error(t, err)
}
