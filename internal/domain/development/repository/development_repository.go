package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"neo360-api/internal/domain/development"
	"neo360-api/internal/pkg/pgerr"
)

const (
	columns = `d.id, d.slug, d.name, d.developer_id, COALESCE(o.name, ''), d.city, d.state, d.status, d.published,
		d.headline, d.description, d.cover_image_url, d.tour_url, d.presentation_url, d.default_layout, d.gallery, d.highlights,
		(SELECT count(*) FROM leads l WHERE l.development_id = d.id), d.created_at, d.updated_at`
	from = ` FROM developments d LEFT JOIN organizations o ON o.id = d.developer_id`

	queryList      = `SELECT ` + columns + from + ` WHERE (NOT $1 OR d.published) ORDER BY d.name`
	queryGet       = `SELECT ` + columns + from + ` WHERE d.id = $1`
	queryBySlug    = `SELECT ` + columns + from + ` WHERE d.slug = $1 AND d.published`
	queryDeveloper = `SELECT EXISTS (SELECT 1 FROM organizations WHERE id = $1 AND kind = 'incorporadora')`
	queryCreate    = `
		INSERT INTO developments (slug, name, developer_id, city, state, status, published, headline, description,
			cover_image_url, tour_url, gallery, highlights, presentation_url, default_layout)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING id`
	queryUpdate = `
		UPDATE developments SET slug = $2, name = $3, developer_id = $4, city = $5, state = $6, status = $7,
			published = $8, headline = $9, description = $10, cover_image_url = $11, tour_url = $12,
			gallery = $13, highlights = $14, presentation_url = $15, default_layout = $16, updated_at = now()
		WHERE id = $1 RETURNING id`

	constraintSlug = "developments_slug_key"
)

type scanner interface {
	Scan(dest ...any) error
}

type DevelopmentRepository struct {
	db *sql.DB
}

func NewDevelopmentRepository(db *sql.DB) *DevelopmentRepository {
	return &DevelopmentRepository{db: db}
}

func (r *DevelopmentRepository) List(ctx context.Context, onlyPublished bool) ([]development.Development, error) {
	rows, err := r.db.QueryContext(ctx, queryList, onlyPublished)
	if err != nil {
		return nil, fmt.Errorf("list developments: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	list := []development.Development{}
	for rows.Next() {
		d, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan development: %w", err)
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list developments: %w", err)
	}
	return list, nil
}

func (r *DevelopmentRepository) Get(ctx context.Context, id string) (development.Development, error) {
	d, err := scan(r.db.QueryRowContext(ctx, queryGet, id))
	return d, notFound(err, "get development")
}

func (r *DevelopmentRepository) GetPublishedBySlug(ctx context.Context, slug string) (development.Development, error) {
	d, err := scan(r.db.QueryRowContext(ctx, queryBySlug, slug))
	return d, notFound(err, "get development by slug")
}

func (r *DevelopmentRepository) Create(ctx context.Context, req development.Request) (development.Development, error) {
	if err := r.checkDeveloper(ctx, req.DeveloperID); err != nil {
		return development.Development{}, err
	}
	args, err := values(req)
	if err != nil {
		return development.Development{}, err
	}
	var id string
	if err := r.db.QueryRowContext(ctx, queryCreate, args...).Scan(&id); err != nil {
		return development.Development{}, writeError(err, "create development")
	}
	return r.Get(ctx, id)
}

func (r *DevelopmentRepository) Update(ctx context.Context, id string, req development.Request) (development.Development, error) {
	if err := r.checkDeveloper(ctx, req.DeveloperID); err != nil {
		return development.Development{}, err
	}
	args, err := values(req)
	if err != nil {
		return development.Development{}, err
	}
	if err := r.db.QueryRowContext(ctx, queryUpdate, append([]any{id}, args...)...).Scan(&id); err != nil {
		return development.Development{}, writeError(err, "update development")
	}
	return r.Get(ctx, id)
}

func (r *DevelopmentRepository) checkDeveloper(ctx context.Context, id *string) error {
	if id == nil {
		return nil
	}
	var ok bool
	if err := r.db.QueryRowContext(ctx, queryDeveloper, *id).Scan(&ok); err != nil {
		if pgerr.BadUUID(err) {
			return development.ErrDeveloperNotFound
		}
		return fmt.Errorf("check developer: %w", err)
	}
	if !ok {
		return development.ErrDeveloperNotFound
	}
	return nil
}

func values(r development.Request) ([]any, error) {
	gallery, err := json.Marshal(r.Gallery)
	if err != nil {
		return nil, fmt.Errorf("gallery: %w", err)
	}
	highlights, err := json.Marshal(r.Highlights)
	if err != nil {
		return nil, fmt.Errorf("highlights: %w", err)
	}
	return []any{r.Slug, r.Name, r.DeveloperID, r.City, r.State, r.Status, r.Published, r.Headline, r.Description,
		r.CoverImageURL, r.TourURL, gallery, highlights, r.PresentationURL, r.DefaultLayout}, nil
}

func scan(s scanner) (development.Development, error) {
	var d development.Development
	var developer sql.NullString
	var gallery, highlights []byte
	err := s.Scan(&d.ID, &d.Slug, &d.Name, &developer, &d.DeveloperName, &d.City, &d.State, &d.Status, &d.Published,
		&d.Headline, &d.Description, &d.CoverImageURL, &d.TourURL, &d.PresentationURL, &d.DefaultLayout, &gallery, &highlights, &d.LeadsCount, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return d, err
	}
	if developer.Valid {
		d.DeveloperID = &developer.String
	}
	if err := json.Unmarshal(gallery, &d.Gallery); err != nil {
		return d, fmt.Errorf("gallery: %w", err)
	}
	if err := json.Unmarshal(highlights, &d.Highlights); err != nil {
		return d, fmt.Errorf("highlights: %w", err)
	}
	return d, nil
}

func writeError(err error, op string) error {
	if pgerr.Unique(err, constraintSlug) {
		return development.ErrSlugTaken
	}
	return notFound(err, op)
}

func notFound(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) || pgerr.BadUUID(err) {
		return development.ErrNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
