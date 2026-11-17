package development

import "context"

type Repository interface {
	List(ctx context.Context, onlyPublished bool) ([]Development, error)
	Get(ctx context.Context, id string) (Development, error)
	GetPublishedBySlug(ctx context.Context, slug string) (Development, error)
	Create(ctx context.Context, r Request) (Development, error)
	Update(ctx context.Context, id string, r Request) (Development, error)
}

type Service interface {
	List(ctx context.Context, onlyPublished bool) ([]Development, error)
	Get(ctx context.Context, id string) (Development, error)
	GetPublishedBySlug(ctx context.Context, slug string) (Development, error)
	Create(ctx context.Context, r Request) (Development, error)
	Update(ctx context.Context, id string, r Request) (Development, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
