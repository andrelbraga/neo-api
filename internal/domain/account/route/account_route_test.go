package route

import (
	"io"
	"log/slog"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neo360-api/internal/config"
	"neo360-api/internal/domain/broker/repository"
	"neo360-api/internal/domain/broker/service"
	"neo360-api/internal/infra/api"
	"neo360-api/internal/pkg/security"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := api.New(config.Configs{JWTSecret: "s", JWTExpiryHours: 1}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	Register(app, service.NewBrokerService(repository.NewBrokerRepository(nil), security.NewBcryptHasher()))
	got := map[string]bool{}
	for _, r := range app.Router().Routes() {
		got[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{"POST /api/auth/login", "POST /api/auth/signup", "GET /api/auth/me", "PUT /api/me/profile",
		"PUT /api/auth/password", "GET /api/admin/users", "POST /api/admin/users/:id/reset-password"} {
		assert.True(t, got[want], want)
	}
}
