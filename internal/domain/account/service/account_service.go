package service

import (
	"context"
	"errors"

	"neo360-api/internal/domain/account"
	"neo360-api/internal/domain/broker"
	"neo360-api/internal/pkg/authz"
)

// dummyHash é comparado quando o e-mail não existe, para o tempo de resposta não revelar quem tem conta.
const dummyHash = "$2a$10$Egt8jl/jVHfM1J/s7ll.sum47MoJ4IUmK2q1UUm97g74fQe7ZCGBS"

type AccountService struct {
	repo     account.Repository
	brokers  account.Brokers
	tokens   account.Tokens
	comparer account.Comparer
}

func NewAccountService(repo account.Repository, brokers account.Brokers, tokens account.Tokens, comparer account.Comparer) *AccountService {
	return &AccountService{repo: repo, brokers: brokers, tokens: tokens, comparer: comparer}
}

func (s *AccountService) Login(ctx context.Context, r account.LoginRequest) (account.Session, error) {
	r = r.Normalize()
	cred, err := s.repo.Credentials(ctx, r.Email)
	if errors.Is(err, account.ErrNotFound) {
		s.comparer.Compare(dummyHash, r.Password)
		return account.Session{}, account.ErrInvalidCredentials
	}
	if err != nil {
		return account.Session{}, err
	}
	if !s.comparer.Compare(cred.PasswordHash, r.Password) || !cred.Active {
		return account.Session{}, account.ErrInvalidCredentials
	}
	return s.session(ctx, cred.UserID, cred.Role)
}

// Signup é o cadastro aberto: a conta nasce como corretor e já pode gerar o link (RN-01.1).
func (s *AccountService) Signup(ctx context.Context, r broker.CreateRequest) (account.Session, error) {
	r.AgencyID = nil
	b, err := s.brokers.Create(ctx, r)
	if err != nil {
		return account.Session{}, err
	}
	return s.session(ctx, b.UserID, authz.RoleBroker)
}

// Me devolve a conta com as permissões e, para o corretor, o cadastro dele.
func (s *AccountService) Me(ctx context.Context, userID string) (account.User, error) {
	u, err := s.repo.User(ctx, userID)
	if err != nil {
		return account.User{}, err
	}
	u.Permissions = authz.Permissions(u.Role)
	if u.Role != authz.RoleBroker {
		return u, nil
	}
	b, err := s.brokers.GetByUser(ctx, userID)
	if err != nil {
		return account.User{}, err
	}
	u.Broker = &b
	return u, nil
}

func (s *AccountService) UpdateProfile(ctx context.Context, userID string, p broker.Profile) (account.User, error) {
	if _, err := s.brokers.UpdateProfile(ctx, userID, p); err != nil {
		return account.User{}, err
	}
	return s.Me(ctx, userID)
}

func (s *AccountService) session(ctx context.Context, userID, role string) (account.Session, error) {
	token, expiresAt, err := s.tokens.Generate(userID, role)
	if err != nil {
		return account.Session{}, err
	}
	u, err := s.Me(ctx, userID)
	if err != nil {
		return account.Session{}, err
	}
	return account.Session{Token: token, ExpiresAt: expiresAt, User: u}, nil
}
