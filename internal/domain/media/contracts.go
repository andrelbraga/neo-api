package media

import (
	"context"

	"neo360-api/internal/infra/storage"
)

// Bucket é o bucket S3 (storage.Bucket).
type Bucket interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Get(ctx context.Context, key string) (storage.Object, error)
	Delete(ctx context.Context, key string) error
}

type Repository interface {
	// SetBrokerPhoto troca a foto do corretor da conta e devolve a chave anterior (vazia se não havia).
	SetBrokerPhoto(ctx context.Context, userID, key string) (old string, err error)
	// BrokerPhotoInUse diz se a chave é a foto atual de algum corretor ativo.
	BrokerPhotoInUse(ctx context.Context, key string) (bool, error)
	// LeadByToken acha o cliente da visita (o que ficou, se foi juntado) pelo hash do token.
	LeadByToken(ctx context.Context, tokenHash string) (string, error)
	CountLeadPhotos(ctx context.Context, leadID string) (int, error)
	AddLeadPhoto(ctx context.Context, leadID string, s Stored, size int) (Photo, error)
	// LeadPhoto acha a foto ligada a uma proposta do cliente; com userID, só se o cliente é do corretor dela.
	LeadPhoto(ctx context.Context, userID, leadID, photoID string) (Stored, error)
}

type Service interface {
	SetBrokerPhoto(ctx context.Context, userID string, data []byte) (BrokerPhoto, error)
	RemoveBrokerPhoto(ctx context.Context, userID string) error
	BrokerPhoto(ctx context.Context, file string) (storage.Object, error)
	UploadLeadPhoto(ctx context.Context, token string, data []byte) (Photo, error)
	// LeadPhoto entrega a foto ao corretor dono do cliente; userID vazio é o admin.
	LeadPhoto(ctx context.Context, userID, leadID, photoID string) (storage.Object, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
