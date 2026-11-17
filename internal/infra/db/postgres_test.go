package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpen_Unreachable(t *testing.T) {
	_, err := Open(context.Background(), "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1", "")
	assert.ErrorContains(t, err, "ping db")
	_, err = Open(context.Background(), "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1", "neo360")
	assert.ErrorContains(t, err, "ping db", "schema válido, banco fora do ar")
}

func TestOpen_InvalidInput(t *testing.T) {
	_, err := Open(context.Background(), "::nao-e-dsn::", "")
	assert.ErrorContains(t, err, "open db")
	for _, bad := range []string{`neo360"; DROP SCHEMA public; --`, "Neo360", "1abc", "neo-360"} {
		_, err = Open(context.Background(), "postgres://u:p@127.0.0.1:1/x?sslmode=disable", bad)
		assert.ErrorIs(t, err, ErrInvalidSchema, bad)
	}
}
