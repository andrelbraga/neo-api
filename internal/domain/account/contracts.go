package account

import (
	"context"
	"time"

	"neo360-api/internal/domain/broker"
)

type Repository interface {
	Credentials(ctx context.Context, email string) (Credentials, error)
	User(ctx context.Context, id string) (User, error)
}

type Service interface {
	Login(ctx context.Context, r LoginRequest) (Session, error)
	Signup(ctx context.Context, r broker.CreateRequest) (Session, error)
	Me(ctx context.Context, userID string) (User, error)
	UpdateProfile(ctx context.Context, userID string, p broker.Profile) (User, error)
}

// Brokers é o que a conta usa do corretor: criar no cadastro aberto, ler e atualizar o perfil.
type Brokers interface {
	Create(ctx context.Context, r broker.CreateRequest) (broker.Broker, error)
	GetByUser(ctx context.Context, userID string) (broker.Broker, error)
	UpdateProfile(ctx context.Context, userID string, p broker.Profile) (broker.Broker, error)
}

type Tokens interface {
	Generate(userID, role string) (string, time.Time, error)
}

type Comparer interface {
	Compare(hash, password string) bool
}

type Logger interface {
	Error(msg string, args ...any)
}
