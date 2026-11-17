package pgerr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestCodes(t *testing.T) {
	unique := fmt.Errorf("wrap: %w", &pgconn.PgError{Code: "23505", ConstraintName: "users_email_key"})
	assert.True(t, Unique(unique, "users_email_key"))
	assert.False(t, Unique(unique, "outra"))
	assert.True(t, ForeignKey(&pgconn.PgError{Code: "23503"}))
	assert.True(t, BadUUID(&pgconn.PgError{Code: "22P02"}))

	plain := errors.New("x")
	assert.False(t, Unique(plain, "users_email_key"))
	assert.False(t, ForeignKey(plain))
	assert.False(t, BadUUID(plain))
}
