package organization

import "context"

type Repository interface {
	List(ctx context.Context, kind string) ([]Organization, error)
	Get(ctx context.Context, id string) (Organization, error)
	Create(ctx context.Context, r Request) (Organization, error)
	Update(ctx context.Context, id string, r Request) (Organization, error)
}

type Service interface {
	List(ctx context.Context, kind string) ([]Organization, error)
	Get(ctx context.Context, id string) (Organization, error)
	Create(ctx context.Context, r Request) (Organization, error)
	Update(ctx context.Context, id string, r Request) (Organization, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
