package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/development"
)

type fakeRepo struct {
	published bool
	page      development.PageConfig
}

func (f *fakeRepo) List(_ context.Context, onlyPublished bool) ([]development.Development, error) {
	f.published = onlyPublished
	return []development.Development{}, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (development.Development, error) {
	return development.Development{ID: id}, nil
}
func (f *fakeRepo) GetPublishedBySlug(_ context.Context, slug string) (development.Development, error) {
	return development.Development{Slug: slug}, nil
}
func (f *fakeRepo) Create(_ context.Context, r development.Request) (development.Development, error) {
	return development.Development{Slug: r.Slug}, nil
}
func (f *fakeRepo) Update(_ context.Context, id string, r development.Request) (development.Development, error) {
	return development.Development{ID: id, Slug: r.Slug}, nil
}

func (f *fakeRepo) SavePage(_ context.Context, id string, c development.PageConfig) (development.Development, error) {
	f.page = c
	return development.Development{ID: id, PageConfig: c}, nil
}

// ADR-0027: a edição da página é limpa antes de gravar.
func TestDevelopmentService_SavePage(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{}
	s := NewDevelopmentService(repo)
	d, err := s.SavePage(ctx, "d1", development.PageConfig{
		Sections: map[string]development.SectionConfig{"h-planta": {Hidden: true}, "h-sim": {Hidden: false}},
		Texts:    []development.TextOverride{{Section: "foot", From: "Protótipo", To: "Valores sujeitos a confirmação."}},
	})
	require.NoError(t, err)
	assert.Equal(t, "d1", d.ID)
	assert.Equal(t, map[string]development.SectionConfig{"h-planta": {Hidden: true}}, repo.page.Sections, "seção sem mudança não é gravada")
	_, err = s.SavePage(ctx, "d1", development.PageConfig{Sections: map[string]development.SectionConfig{"<script>": {Hidden: true}}})
	assert.ErrorIs(t, err, development.ErrInvalidPage)
}

func TestDevelopmentService(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{}
	s := NewDevelopmentService(repo)

	_, err := s.List(ctx, true)
	require.NoError(t, err)
	assert.True(t, repo.published)

	d, err := s.Get(ctx, "d1")
	require.NoError(t, err)
	assert.Equal(t, "d1", d.ID)
	d, err = s.GetPublishedBySlug(ctx, "boa-vista")
	require.NoError(t, err)
	assert.Equal(t, "boa-vista", d.Slug)

	d, err = s.Create(ctx, development.Request{Name: "Boa Vista"})
	require.NoError(t, err)
	assert.Equal(t, "boa-vista", d.Slug)
	_, err = s.Create(ctx, development.Request{})
	assert.ErrorIs(t, err, development.ErrNameRequired)

	d, err = s.Update(ctx, "d1", development.Request{Name: "Boa Vista", Slug: "bv"})
	require.NoError(t, err)
	assert.Equal(t, "bv", d.Slug)
	_, err = s.Update(ctx, "d1", development.Request{Name: "x", Status: "?"})
	assert.ErrorIs(t, err, development.ErrInvalidStatus)
}
