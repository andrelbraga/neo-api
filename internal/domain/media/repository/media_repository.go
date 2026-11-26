package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"neo360-api/internal/domain/media"
	"neo360-api/internal/pkg/pgerr"
)

const (
	// A foto anterior sai na mesma consulta, para o serviço apagar do bucket.
	querySetBrokerPhoto = `
		UPDATE brokers b SET photo_key = $2, updated_at = now()
		FROM (SELECT id, photo_key FROM brokers WHERE user_id = $1 FOR UPDATE) old
		WHERE b.id = old.id RETURNING old.photo_key`
	queryBrokerPhotoInUse = `
		SELECT EXISTS (SELECT 1 FROM brokers b JOIN users u ON u.id = b.user_id WHERE b.photo_key = $1 AND u.active)`
	// Token de um cadastro juntado a outro (ADR-0016) vale pelo cadastro que ficou.
	queryLeadByToken = `SELECT COALESCE(merged_into, id) FROM leads WHERE token_hash = $1`
	queryCountPhotos = `SELECT count(*) FROM lead_photos WHERE lead_id = $1`
	queryAddPhoto    = `INSERT INTO lead_photos (lead_id, object_key, content_type, size_bytes) VALUES ($1, $2, $3, $4) RETURNING id`
	// Só a foto ligada a uma proposta do cliente; com $3, só se o cliente é do corretor da conta.
	queryLeadPhoto = `
		SELECT p.object_key, p.content_type
		FROM lead_photos p JOIN proposals pr ON pr.id = p.proposal_id JOIN leads l ON l.id = pr.lead_id
		LEFT JOIN brokers b ON b.id = l.broker_id
		WHERE p.id = $1 AND pr.lead_id = $2 AND ($3 = '' OR b.user_id::text = $3)`
)

type MediaRepository struct {
	db *sql.DB
}

func NewMediaRepository(db *sql.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

func (r *MediaRepository) SetBrokerPhoto(ctx context.Context, userID, key string) (string, error) {
	var old string
	err := r.db.QueryRowContext(ctx, querySetBrokerPhoto, userID, key).Scan(&old)
	if errors.Is(err, sql.ErrNoRows) || pgerr.BadUUID(err) {
		return "", media.ErrBrokerNotFound
	}
	if err != nil {
		return "", fmt.Errorf("set broker photo: %w", err)
	}
	return old, nil
}

func (r *MediaRepository) BrokerPhotoInUse(ctx context.Context, key string) (bool, error) {
	var ok bool
	if err := r.db.QueryRowContext(ctx, queryBrokerPhotoInUse, key).Scan(&ok); err != nil {
		return false, fmt.Errorf("broker photo in use: %w", err)
	}
	return ok, nil
}

func (r *MediaRepository) LeadByToken(ctx context.Context, tokenHash string) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, queryLeadByToken, tokenHash).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", media.ErrInvalidToken
	}
	if err != nil {
		return "", fmt.Errorf("lead by token: %w", err)
	}
	return id, nil
}

func (r *MediaRepository) CountLeadPhotos(ctx context.Context, leadID string) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, queryCountPhotos, leadID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count lead photos: %w", err)
	}
	return n, nil
}

func (r *MediaRepository) AddLeadPhoto(ctx context.Context, leadID string, s media.Stored, size int) (media.Photo, error) {
	p := media.Photo{ContentType: s.ContentType, Size: size}
	if err := r.db.QueryRowContext(ctx, queryAddPhoto, leadID, s.Key, s.ContentType, size).Scan(&p.ID); err != nil {
		return p, fmt.Errorf("add lead photo: %w", err)
	}
	return p, nil
}

func (r *MediaRepository) LeadPhoto(ctx context.Context, userID, leadID, photoID string) (media.Stored, error) {
	var s media.Stored
	err := r.db.QueryRowContext(ctx, queryLeadPhoto, photoID, leadID, userID).Scan(&s.Key, &s.ContentType)
	if errors.Is(err, sql.ErrNoRows) || pgerr.BadUUID(err) {
		return s, media.ErrNotFound
	}
	if err != nil {
		return s, fmt.Errorf("lead photo: %w", err)
	}
	return s, nil
}
