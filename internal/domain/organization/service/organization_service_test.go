package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/organization"
)

type fakeRepo struct{ kind string }

func (f *fakeRepo) List(_ context.Context, kind string) ([]organization.Organization, error) {
	f.kind = kind
	return []organization.Organization{}, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (organization.Organization, error) {
	return organization.Organization{ID: id}, nil
}
func (f *fakeRepo) Create(_ context.Context, r organization.Request) (organization.Organization, error) {
	return organization.Organization{Name: r.Name}, nil
}
func (f *fakeRepo) Update(_ context.Context, id string, r organization.Request) (organization.Organization, error) {
	return organization.Organization{ID: id, Name: r.Name}, nil
}

func TestOrganizationService(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{}
	s := NewOrganizationService(repo)

	_, err := s.List(ctx, " Incorporadora ")
	require.NoError(t, err)
	assert.Equal(t, organization.KindDeveloper, repo.kind)
	_, err = s.List(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, repo.kind, "sem tipo lista todas")
	_, err = s.List(ctx, "construtora")
	assert.ErrorIs(t, err, organization.ErrInvalidKind)

	o, err := s.Get(ctx, "o1")
	require.NoError(t, err)
	assert.Equal(t, "o1", o.ID)

	o, err = s.Create(ctx, organization.Request{Kind: organization.KindAgency, Name: " Omni "})
	require.NoError(t, err)
	assert.Equal(t, "Omni", o.Name)
	_, err = s.Create(ctx, organization.Request{})
	assert.ErrorIs(t, err, organization.ErrInvalidKind)

	o, err = s.Update(ctx, "o1", organization.Request{Kind: organization.KindAgency, Name: "Omni"})
	require.NoError(t, err)
	assert.Equal(t, "o1", o.ID)
	_, err = s.Update(ctx, "o1", organization.Request{Kind: organization.KindAgency})
	assert.ErrorIs(t, err, organization.ErrNameRequired)
}
