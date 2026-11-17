package service

import (
	"context"
	"errors"
	"math/rand/v2"

	"neo360-api/internal/domain/broker"
)

// codeAttempts é quantas vezes o cadastro sorteia outro código antes de desistir.
const codeAttempts = 5

type BrokerService struct {
	repo   broker.Repository
	hasher broker.Hasher
	random func() int
}

func NewBrokerService(repo broker.Repository, hasher broker.Hasher) *BrokerService {
	return &BrokerService{repo: repo, hasher: hasher, random: func() int { return rand.IntN(10000) }}
}

// Create cadastra a conta e o corretor. O código do link é sorteado e, se já existir, sorteado de novo.
func (s *BrokerService) Create(ctx context.Context, r broker.CreateRequest) (broker.Broker, error) {
	valid, err := r.Validate()
	if err != nil {
		return broker.Broker{}, err
	}
	hash, err := s.hasher.Hash(valid.Password)
	if err != nil {
		return broker.Broker{}, err
	}
	for range codeAttempts {
		b, err := s.repo.Create(ctx, valid, hash, broker.NewCode(valid.FirstName, s.random()))
		if !errors.Is(err, broker.ErrCodeTaken) {
			return b, err
		}
	}
	return broker.Broker{}, broker.ErrCodeTaken
}

func (s *BrokerService) List(ctx context.Context) ([]broker.Broker, error) {
	return s.repo.List(ctx)
}

func (s *BrokerService) Get(ctx context.Context, id string) (broker.Broker, error) {
	return s.repo.Get(ctx, id)
}

func (s *BrokerService) GetByUser(ctx context.Context, userID string) (broker.Broker, error) {
	return s.repo.GetByUser(ctx, userID)
}

func (s *BrokerService) Update(ctx context.Context, id string, r broker.UpdateRequest) (broker.Broker, error) {
	valid, err := r.Validate()
	if err != nil {
		return broker.Broker{}, err
	}
	return s.repo.Update(ctx, id, valid)
}

func (s *BrokerService) UpdateProfile(ctx context.Context, userID string, p broker.Profile) (broker.Broker, error) {
	valid, err := p.Validate()
	if err != nil {
		return broker.Broker{}, err
	}
	return s.repo.UpdateProfile(ctx, userID, valid)
}
