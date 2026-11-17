package route

import (
	"io"
	"log/slog"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neo360-api/internal/config"
	"neo360-api/internal/domain/development/repository"
	"neo360-api/internal/domain/development/service"
	"neo360-api/internal/infra/api"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := api.New(config.Configs{JWTSecret: "s", JWTExpiryHours: 1}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	Register(app, service.NewDevelopmentService(repository.NewDevelopmentRepository(nil)))
	got := map[string]bool{}
	for _, r := range app.Router().Routes() {
		got[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{"GET /api/admin/developments", "POST /api/admin/developments", "GET /api/admin/developments/:id",
		"PUT /api/admin/developments/:id", "GET /api/me/developments"} {
		assert.True(t, got[want], want)
	}
}
