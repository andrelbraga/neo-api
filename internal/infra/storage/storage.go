// Package storage guarda arquivos no bucket S3 do projeto (Railway Bucket neo-files): a foto do corretor e as
// fotos dos veículos da proposta. O bucket é privado; quem entrega o arquivo ao navegador é a API.
package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"neo360-api/internal/config"
)

// ErrNotFound é o arquivo que não existe no bucket.
var ErrNotFound = errors.New("arquivo não encontrado")

// Object é o arquivo lido do bucket. Quem lê fecha o Body.
type Object struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
}

// Bucket é o bucket S3. Os domínios usam por interface (Put, Get, Delete).
type Bucket struct {
	client *minio.Client
	name   string
}

// New conecta ao bucket. Sem configuração, devolve nil: o envio de fotos fica desligado.
func New(cfg config.Storage) (*Bucket, error) {
	if !cfg.Enabled() {
		return nil, nil
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("storage endpoint: %q inválido", cfg.Endpoint)
	}
	client, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: u.Scheme != "http",
		Region: cfg.Region,
		// O Railway usa o nome do bucket no subdomínio (virtual-hosted style).
		BucketLookup: minio.BucketLookupDNS,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: %w", err)
	}
	return &Bucket{client: client, name: cfg.Bucket}, nil
}

func (b *Bucket) Put(ctx context.Context, key, contentType string, data []byte) error {
	_, err := b.client.PutObject(ctx, b.name, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func (b *Bucket) Get(ctx context.Context, key string) (Object, error) {
	obj, err := b.client.GetObject(ctx, b.name, key, minio.GetObjectOptions{})
	if err != nil {
		return Object{}, fmt.Errorf("get object: %w", err)
	}
	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		if minio.ToErrorResponse(err).StatusCode == http.StatusNotFound {
			return Object{}, ErrNotFound
		}
		return Object{}, fmt.Errorf("stat object: %w", err)
	}
	return Object{Body: obj, ContentType: info.ContentType, Size: info.Size}, nil
}

func (b *Bucket) Delete(ctx context.Context, key string) error {
	if err := b.client.RemoveObject(ctx, b.name, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

// Image confere pelo conteúdo (não pelo nome nem pelo tipo informado) que o arquivo é JPEG, PNG ou WebP, e
// devolve o tipo e a extensão a usar no bucket.
func Image(data []byte) (contentType, ext string, ok bool) {
	ct := http.DetectContentType(data)
	switch {
	case ct == "image/jpeg":
		return ct, "jpg", true
	case ct == "image/png":
		return ct, "png", true
	case ct == "image/webp" || (len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"):
		return "image/webp", "webp", true
	}
	return strings.TrimSpace(ct), "", false
}
