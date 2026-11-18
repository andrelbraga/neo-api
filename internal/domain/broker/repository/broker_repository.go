package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"neo360-api/internal/domain/broker"
	"neo360-api/internal/pkg/pgerr"
)

const (
	columns = `b.id, b.user_id, b.first_name, b.last_name, u.email, b.phone, b.creci, b.code, b.agency_id,
		COALESCE(o.name, ''), COALESCE(o.logo_url, ''), u.active, (SELECT count(*) FROM leads l WHERE l.broker_id = b.id), b.created_at`
	from = ` FROM brokers b JOIN users u ON u.id = b.user_id LEFT JOIN organizations o ON o.id = b.agency_id`

	queryList      = `SELECT ` + columns + from + ` ORDER BY b.first_name, b.last_name`
	queryGet       = `SELECT ` + columns + from + ` WHERE b.id = $1`
	queryGetByUser = `SELECT ` + columns + from + ` WHERE b.user_id = $1`
	queryAgency    = `SELECT EXISTS (SELECT 1 FROM organizations WHERE id = $1 AND kind = 'imobiliaria')`
	queryUser      = `INSERT INTO users (name, email, password_hash, role) VALUES ($1, $2, $3, 'corretor') RETURNING id`
	queryBroker    = `
		INSERT INTO brokers (user_id, first_name, last_name, phone, creci, code, agency_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	queryUpdate = `
		UPDATE brokers SET first_name = $2, last_name = $3, phone = $4, creci = $5, agency_id = $6, updated_at = now()
		WHERE id = $1 RETURNING user_id`
	queryUpdateProfile = `
		UPDATE brokers SET first_name = $2, last_name = $3, phone = $4, creci = $5, updated_at = now()
		WHERE user_id = $1 RETURNING id`
	queryUserSync  = `UPDATE users SET name = $2, active = COALESCE($3, active), updated_at = now() WHERE id = $1`
	queryUserEmail = `UPDATE users SET email = $2, updated_at = now() WHERE id = $1`

	constraintEmail = "users_email_key"
	constraintCode  = "brokers_code_key"
)

type scanner interface {
	Scan(dest ...any) error
}

type BrokerRepository struct {
	db *sql.DB
}

func NewBrokerRepository(db *sql.DB) *BrokerRepository {
	return &BrokerRepository{db: db}
}

// Create grava a conta e o corretor na mesma transação.
func (r *BrokerRepository) Create(ctx context.Context, req broker.CreateRequest, hash, code string) (broker.Broker, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return broker.Broker{}, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := checkAgency(ctx, tx, req.AgencyID); err != nil {
		return broker.Broker{}, err
	}
	var userID, id string
	name := broker.Broker{FirstName: req.FirstName, LastName: req.LastName}.FullName()
	if err := tx.QueryRowContext(ctx, queryUser, name, req.Email, hash).Scan(&userID); err != nil {
		if pgerr.Unique(err, constraintEmail) {
			return broker.Broker{}, broker.ErrEmailTaken
		}
		return broker.Broker{}, fmt.Errorf("create user: %w", err)
	}
	err = tx.QueryRowContext(ctx, queryBroker, userID, req.FirstName, req.LastName, req.Phone, req.Creci, code, req.AgencyID).Scan(&id)
	if err != nil {
		if pgerr.Unique(err, constraintCode) {
			return broker.Broker{}, broker.ErrCodeTaken
		}
		return broker.Broker{}, fmt.Errorf("create broker: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return broker.Broker{}, fmt.Errorf("commit: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *BrokerRepository) List(ctx context.Context) ([]broker.Broker, error) {
	rows, err := r.db.QueryContext(ctx, queryList)
	if err != nil {
		return nil, fmt.Errorf("list brokers: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	list := []broker.Broker{}
	for rows.Next() {
		b, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan broker: %w", err)
		}
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list brokers: %w", err)
	}
	return list, nil
}

func (r *BrokerRepository) Get(ctx context.Context, id string) (broker.Broker, error) {
	b, err := scan(r.db.QueryRowContext(ctx, queryGet, id))
	return b, notFound(err, "get broker")
}

func (r *BrokerRepository) GetByUser(ctx context.Context, userID string) (broker.Broker, error) {
	b, err := scan(r.db.QueryRowContext(ctx, queryGetByUser, userID))
	return b, notFound(err, "get broker by user")
}

// Update é a edição pelo admin: perfil, imobiliária e se a conta está ativa (RN-01.5).
func (r *BrokerRepository) Update(ctx context.Context, id string, req broker.UpdateRequest) (broker.Broker, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return broker.Broker{}, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := checkAgency(ctx, tx, req.AgencyID); err != nil {
		return broker.Broker{}, err
	}
	var userID string
	err = tx.QueryRowContext(ctx, queryUpdate, id, req.FirstName, req.LastName, req.Phone, req.Creci, req.AgencyID).Scan(&userID)
	if err != nil {
		return broker.Broker{}, notFound(err, "update broker")
	}
	name := broker.Broker{FirstName: req.FirstName, LastName: req.LastName}.FullName()
	if _, err := tx.ExecContext(ctx, queryUserSync, userID, name, req.Active); err != nil {
		return broker.Broker{}, fmt.Errorf("sync user: %w", err)
	}
	if req.Email != "" {
		if _, err := tx.ExecContext(ctx, queryUserEmail, userID, req.Email); err != nil {
			if pgerr.Unique(err, constraintEmail) {
				return broker.Broker{}, broker.ErrEmailTaken
			}
			return broker.Broker{}, fmt.Errorf("update email: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return broker.Broker{}, fmt.Errorf("commit: %w", err)
	}
	return r.Get(ctx, id)
}

// UpdateProfile é a edição pelo próprio corretor; não mexe na imobiliária nem na situação da conta.
func (r *BrokerRepository) UpdateProfile(ctx context.Context, userID string, p broker.Profile) (broker.Broker, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return broker.Broker{}, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var id string
	if err := tx.QueryRowContext(ctx, queryUpdateProfile, userID, p.FirstName, p.LastName, p.Phone, p.Creci).Scan(&id); err != nil {
		return broker.Broker{}, notFound(err, "update profile")
	}
	name := broker.Broker{FirstName: p.FirstName, LastName: p.LastName}.FullName()
	if _, err := tx.ExecContext(ctx, queryUserSync, userID, name, nil); err != nil {
		return broker.Broker{}, fmt.Errorf("sync user: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return broker.Broker{}, fmt.Errorf("commit: %w", err)
	}
	return r.Get(ctx, id)
}

func checkAgency(ctx context.Context, tx *sql.Tx, agencyID *string) error {
	if agencyID == nil {
		return nil
	}
	var ok bool
	if err := tx.QueryRowContext(ctx, queryAgency, *agencyID).Scan(&ok); err != nil {
		if pgerr.BadUUID(err) {
			return broker.ErrAgencyNotFound
		}
		return fmt.Errorf("check agency: %w", err)
	}
	if !ok {
		return broker.ErrAgencyNotFound
	}
	return nil
}

func scan(s scanner) (broker.Broker, error) {
	var b broker.Broker
	var agency sql.NullString
	err := s.Scan(&b.ID, &b.UserID, &b.FirstName, &b.LastName, &b.Email, &b.Phone, &b.Creci, &b.Code, &agency,
		&b.AgencyName, &b.AgencyLogo, &b.Active, &b.LeadsCount, &b.CreatedAt)
	if agency.Valid {
		b.AgencyID = &agency.String
	}
	return b, err
}

func notFound(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) || pgerr.BadUUID(err) {
		return broker.ErrNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
