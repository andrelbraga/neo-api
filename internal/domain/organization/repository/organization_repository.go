package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"neo360-api/internal/domain/organization"
	"neo360-api/internal/pkg/pgerr"
)

const (
	columns     = `id, kind, name, document, creci, phone, email, website, logo_url, city, active, created_at`
	queryList   = `SELECT ` + columns + ` FROM organizations WHERE ($1 = '' OR kind = $1) ORDER BY city, name`
	queryGet    = `SELECT ` + columns + ` FROM organizations WHERE id = $1`
	queryCreate = `
		INSERT INTO organizations (kind, name, document, creci, phone, email, website, active, logo_url, city)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING ` + columns
	queryUpdate = `
		UPDATE organizations SET kind = $2, name = $3, document = $4, creci = $5, phone = $6, email = $7,
			website = $8, active = $9, logo_url = $10, city = $11, updated_at = now()
		WHERE id = $1 RETURNING ` + columns
)

type scanner interface {
	Scan(dest ...any) error
}

type OrganizationRepository struct {
	db *sql.DB
}

func NewOrganizationRepository(db *sql.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

func (r *OrganizationRepository) List(ctx context.Context, kind string) ([]organization.Organization, error) {
	rows, err := r.db.QueryContext(ctx, queryList, kind)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	list := []organization.Organization{}
	for rows.Next() {
		o, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		list = append(list, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	return list, nil
}

func (r *OrganizationRepository) Get(ctx context.Context, id string) (organization.Organization, error) {
	o, err := scan(r.db.QueryRowContext(ctx, queryGet, id))
	return o, notFound(err, "get organization")
}

func (r *OrganizationRepository) Create(ctx context.Context, req organization.Request) (organization.Organization, error) {
	o, err := scan(r.db.QueryRowContext(ctx, queryCreate, req.Kind, req.Name, req.Document, req.Creci, req.Phone,
		req.Email, req.Website, req.Active, req.LogoURL, req.City))
	if err != nil {
		return o, fmt.Errorf("create organization: %w", err)
	}
	return o, nil
}

func (r *OrganizationRepository) Update(ctx context.Context, id string, req organization.Request) (organization.Organization, error) {
	o, err := scan(r.db.QueryRowContext(ctx, queryUpdate, id, req.Kind, req.Name, req.Document, req.Creci, req.Phone,
		req.Email, req.Website, req.Active, req.LogoURL, req.City))
	return o, notFound(err, "update organization")
}

func scan(s scanner) (organization.Organization, error) {
	var o organization.Organization
	err := s.Scan(&o.ID, &o.Kind, &o.Name, &o.Document, &o.Creci, &o.Phone, &o.Email, &o.Website, &o.LogoURL, &o.City, &o.Active, &o.CreatedAt)
	return o, err
}

func notFound(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) || pgerr.BadUUID(err) {
		return organization.ErrNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
