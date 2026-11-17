package service

import (
	"context"
	"strings"

	"neo360-api/internal/domain/organization"
)

type OrganizationService struct {
	repo organization.Repository
}

func NewOrganizationService(repo organization.Repository) *OrganizationService {
	return &OrganizationService{repo: repo}
}

// List filtra pelo tipo quando informado; tipo vazio lista todas.
func (s *OrganizationService) List(ctx context.Context, kind string) ([]organization.Organization, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind != "" && !organization.ValidKind(kind) {
		return nil, organization.ErrInvalidKind
	}
	return s.repo.List(ctx, kind)
}

func (s *OrganizationService) Get(ctx context.Context, id string) (organization.Organization, error) {
	return s.repo.Get(ctx, id)
}

func (s *OrganizationService) Create(ctx context.Context, r organization.Request) (organization.Organization, error) {
	valid, err := r.Validate()
	if err != nil {
		return organization.Organization{}, err
	}
	return s.repo.Create(ctx, valid)
}

func (s *OrganizationService) Update(ctx context.Context, id string, r organization.Request) (organization.Organization, error) {
	valid, err := r.Validate()
	if err != nil {
		return organization.Organization{}, err
	}
	return s.repo.Update(ctx, id, valid)
}
