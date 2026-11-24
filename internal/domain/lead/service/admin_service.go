package service

import (
	"context"

	"neo360-api/internal/domain/lead"
)

// AdminService é a visão de todos os clientes no admin. Abrir a ficha aqui não marca as propostas como vistas.
type AdminService struct {
	repo lead.Repository
}

func NewAdminService(repo lead.Repository) *AdminService {
	return &AdminService{repo: repo}
}

func (s *AdminService) List(ctx context.Context, f lead.Filter) ([]lead.Lead, error) {
	if f.Stage != "" && !lead.ValidStage(f.Stage) {
		return nil, lead.ErrInvalidStage
	}
	return s.repo.List(ctx, f)
}

func (s *AdminService) Get(ctx context.Context, id string) (lead.Detail, error) {
	l, err := s.repo.Get(ctx, id)
	if err != nil {
		return lead.Detail{}, err
	}
	return detail(ctx, s.repo, l)
}

// Summary soma tudo, ou só os clientes de um corretor quando o admin vê como ele (RN-01.10).
func (s *AdminService) Summary(ctx context.Context, brokerID string) (lead.Summary, error) {
	return s.repo.Summary(ctx, brokerID)
}

// Activity é o feed de todos, ou só dos clientes de um corretor quando o admin vê como ele (RN-01.10).
func (s *AdminService) Activity(ctx context.Context, brokerID string) ([]lead.Activity, error) {
	return s.repo.Activity(ctx, brokerID)
}
