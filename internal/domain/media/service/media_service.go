package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"neo360-api/internal/domain/media"
	"neo360-api/internal/infra/storage"
)

type MediaService struct {
	repo   media.Repository
	bucket media.Bucket
	log    media.Logger
}

// NewMediaService recebe o bucket já conectado; nil desliga o envio e a leitura de fotos.
func NewMediaService(repo media.Repository, bucket media.Bucket, log media.Logger) *MediaService {
	return &MediaService{repo: repo, bucket: bucket, log: log}
}

// SetBrokerPhoto guarda a foto nova, aponta o corretor para ela e apaga a anterior (RN-01.11).
func (s *MediaService) SetBrokerPhoto(ctx context.Context, userID string, data []byte) (media.BrokerPhoto, error) {
	stored, err := s.image(data, media.MaxBrokerPhotoBytes, media.BrokerPrefix)
	if err != nil {
		return media.BrokerPhoto{}, err
	}
	if err := s.bucket.Put(ctx, stored.Key, stored.ContentType, data); err != nil {
		return media.BrokerPhoto{}, err
	}
	old, err := s.repo.SetBrokerPhoto(ctx, userID, stored.Key)
	if err != nil {
		s.remove(ctx, stored.Key)
		return media.BrokerPhoto{}, err
	}
	s.remove(ctx, old)
	return media.BrokerPhoto{PhotoPath: media.BrokerPhotoPath(stored.Key)}, nil
}

// RemoveBrokerPhoto volta às iniciais no cartão.
func (s *MediaService) RemoveBrokerPhoto(ctx context.Context, userID string) error {
	old, err := s.repo.SetBrokerPhoto(ctx, userID, "")
	if err != nil {
		return err
	}
	if s.bucket != nil {
		s.remove(ctx, old)
	}
	return nil
}

// BrokerPhoto entrega a foto atual de um corretor; foto trocada ou de corretor desativado não sai.
func (s *MediaService) BrokerPhoto(ctx context.Context, file string) (storage.Object, error) {
	key := media.BrokerKey(file)
	if key == "" || s.bucket == nil {
		return storage.Object{}, media.ErrNotFound
	}
	ok, err := s.repo.BrokerPhotoInUse(ctx, key)
	if err != nil {
		return storage.Object{}, err
	}
	if !ok {
		return storage.Object{}, media.ErrNotFound
	}
	return s.get(ctx, key)
}

// UploadLeadPhoto recebe a foto de um veículo da proposta (RN-03.23). Ela só aparece para o corretor depois
// que o cliente envia a proposta com o id dela.
func (s *MediaService) UploadLeadPhoto(ctx context.Context, token string, data []byte) (media.Photo, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return media.Photo{}, media.ErrInvalidToken
	}
	if s.bucket == nil {
		return media.Photo{}, media.ErrUnavailable
	}
	leadID, err := s.repo.LeadByToken(ctx, hash(token))
	if err != nil {
		return media.Photo{}, err
	}
	n, err := s.repo.CountLeadPhotos(ctx, leadID)
	if err != nil {
		return media.Photo{}, err
	}
	if n >= media.MaxPhotosPerLead {
		return media.Photo{}, media.ErrTooMany
	}
	stored, err := s.image(data, media.MaxVehiclePhotoBytes, media.LeadPrefix+leadID+"/")
	if err != nil {
		return media.Photo{}, err
	}
	if err := s.bucket.Put(ctx, stored.Key, stored.ContentType, data); err != nil {
		return media.Photo{}, err
	}
	p, err := s.repo.AddLeadPhoto(ctx, leadID, stored, len(data))
	if err != nil {
		s.remove(ctx, stored.Key)
		return media.Photo{}, err
	}
	return p, nil
}

func (s *MediaService) LeadPhoto(ctx context.Context, userID, leadID, photoID string) (storage.Object, error) {
	if s.bucket == nil {
		return storage.Object{}, media.ErrNotFound
	}
	stored, err := s.repo.LeadPhoto(ctx, userID, leadID, photoID)
	if err != nil {
		return storage.Object{}, err
	}
	return s.get(ctx, stored.Key)
}

// image confere a foto e escolhe a chave no bucket: nome aleatório, que também serve de cache permanente.
func (s *MediaService) image(data []byte, limit int, prefix string) (media.Stored, error) {
	if s.bucket == nil {
		return media.Stored{}, media.ErrUnavailable
	}
	if len(data) > limit {
		return media.Stored{}, media.ErrTooLarge
	}
	ct, ext, ok := storage.Image(data)
	if len(data) == 0 || !ok {
		return media.Stored{}, media.ErrNotImage
	}
	name, err := randomName()
	if err != nil {
		return media.Stored{}, err
	}
	return media.Stored{Key: prefix + name + "." + ext, ContentType: ct}, nil
}

func (s *MediaService) get(ctx context.Context, key string) (storage.Object, error) {
	obj, err := s.bucket.Get(ctx, key)
	if errors.Is(err, storage.ErrNotFound) {
		return obj, media.ErrNotFound
	}
	return obj, err
}

// remove apaga um arquivo que deixou de ser usado; falha não volta ao usuário (o arquivo só sobra no bucket).
func (s *MediaService) remove(ctx context.Context, key string) {
	if key == "" {
		return
	}
	if err := s.bucket.Delete(ctx, key); err != nil {
		s.log.Error("delete photo failed", "error", err)
	}
}

func randomName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("photo name: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// hash é o mesmo do token da visita no showroom (lead/service): o banco guarda só o hash (RN-03.2).
func hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
