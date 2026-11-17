package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

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
	comparer account.Hasher
	newTemp  func() (string, error)
}

func NewAccountService(repo account.Repository, brokers account.Brokers, tokens account.Tokens, hasher account.Hasher) *AccountService {
	return &AccountService{repo: repo, brokers: brokers, tokens: tokens, comparer: hasher, newTemp: temporaryPassword}
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

// ChangePassword troca a senha do próprio usuário: confere a atual (ou a provisória) e grava a nova. Com isso
// a troca obrigatória sai e o resto do sistema volta a responder (ADR-0020).
func (s *AccountService) ChangePassword(ctx context.Context, userID string, r account.PasswordChange) (account.User, error) {
	if err := r.Validate(); err != nil {
		return account.User{}, err
	}
	hash, err := s.repo.PasswordHash(ctx, userID)
	if err != nil {
		return account.User{}, err
	}
	if !s.comparer.Compare(hash, r.Current) {
		return account.User{}, account.ErrWrongPassword
	}
	newHash, err := s.comparer.Hash(r.New)
	if err != nil {
		return account.User{}, err
	}
	if err := s.repo.ChangePassword(ctx, userID, newHash); err != nil {
		return account.User{}, err
	}
	return s.Me(ctx, userID)
}

// ListAccounts é a lista de usuários do admin. Não traz senha: ela só existe como hash, que não se desfaz.
func (s *AccountService) ListAccounts(ctx context.Context) ([]account.Account, error) {
	return s.repo.ListAccounts(ctx)
}

// ResetPassword gera uma senha provisória, grava só o hash dela e marca a troca no primeiro acesso.
// A senha provisória volta uma única vez, para o admin passar ao usuário (ADR-0020).
func (s *AccountService) ResetPassword(ctx context.Context, userID string) (account.TemporaryPassword, error) {
	temp, err := s.newTemp()
	if err != nil {
		return account.TemporaryPassword{}, err
	}
	hash, err := s.comparer.Hash(temp)
	if err != nil {
		return account.TemporaryPassword{}, err
	}
	if err := s.repo.ResetPassword(ctx, userID, hash); err != nil {
		return account.TemporaryPassword{}, err
	}
	return account.TemporaryPassword{Password: temp}, nil
}

// alfabeto sem letras e números que se confundem (0/O, 1/l/I), para ditar ou digitar a senha provisória.
const tempAlphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// temporaryPassword gera 3 blocos de 4 caracteres (ex.: "k7Hm-2pQx-9sWd"), com fonte aleatória segura.
func temporaryPassword() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("temporary password: %w", err)
	}
	out := make([]byte, 0, 14)
	for i, v := range b {
		if i > 0 && i%4 == 0 {
			out = append(out, '-')
		}
		out = append(out, tempAlphabet[int(v)%len(tempAlphabet)])
	}
	return string(out), nil
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
