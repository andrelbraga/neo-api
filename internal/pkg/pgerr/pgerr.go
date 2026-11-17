// Package pgerr traduz os códigos de erro do PostgreSQL que os repositórios tratam.
package pgerr

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	uniqueViolation           = "23505"
	foreignKeyViolation       = "23503"
	invalidTextRepresentation = "22P02"
)

// Unique diz se err é violação da restrição única informada.
func Unique(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation && pgErr.ConstraintName == constraint
}

// ForeignKey diz se err é referência a um registro que não existe.
func ForeignKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation
}

// BadUUID diz se err vem de um id que não é UUID: para quem chama, é um registro que não existe.
func BadUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation
}
