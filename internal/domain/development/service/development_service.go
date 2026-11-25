package service

import (
	"context"

	"neo360-api/internal/domain/development"
)

type DevelopmentService struct {
	repo development.Repository
}

func NewDevelopmentService(repo development.Repository) *DevelopmentService {
	return &DevelopmentService{repo: repo}
}

func (s *DevelopmentService) List(ctx context.Context, onlyPublished bool) ([]development.Development, error) {
	return s.repo.List(ctx, onlyPublished)
}

func (s *DevelopmentService) Get(ctx context.Context, id string) (development.Development, error) {
	return s.repo.Get(ctx, id)
}

func (s *DevelopmentService) GetPublishedBySlug(ctx context.Context, slug string) (development.Development, error) {
	return s.repo.GetPublishedBySlug(ctx, slug)
}

func (s *DevelopmentService) Create(ctx context.Context, r development.Request) (development.Development, error) {
	valid, err := r.Validate()
	if err != nil {
		return development.Development{}, err
	}
	return s.repo.Create(ctx, valid)
}

func (s *DevelopmentService) Update(ctx context.Context, id string, r development.Request) (development.Development, error) {
	valid, err := r.Validate()
	if err != nil {
		return development.Development{}, err
	}
	return s.repo.Update(ctx, id, valid)
}

func (s *DevelopmentService) SavePage(ctx context.Context, id string, c development.PageConfig) (development.Development, error) {
	valid, err := c.Validate()
	if err != nil {
		return development.Development{}, err
	}
	return s.repo.SavePage(ctx, id, valid)
}
