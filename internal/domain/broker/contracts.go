package broker

import "context"

type Repository interface {
	Create(ctx context.Context, r CreateRequest, passwordHash, code string) (Broker, error)
	List(ctx context.Context) ([]Broker, error)
	Get(ctx context.Context, id string) (Broker, error)
	GetByUser(ctx context.Context, userID string) (Broker, error)
	Update(ctx context.Context, id string, r UpdateRequest) (Broker, error)
	UpdateProfile(ctx context.Context, userID string, p Profile) (Broker, error)
}

type Service interface {
	Create(ctx context.Context, r CreateRequest) (Broker, error)
	List(ctx context.Context) ([]Broker, error)
	Get(ctx context.Context, id string) (Broker, error)
	GetByUser(ctx context.Context, userID string) (Broker, error)
	Update(ctx context.Context, id string, r UpdateRequest) (Broker, error)
	UpdateProfile(ctx context.Context, userID string, p Profile) (Broker, error)
}

type Hasher interface {
	Hash(password string) (string, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
