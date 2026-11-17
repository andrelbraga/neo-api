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
	for _, want := range []string{
		"GET /api/public/showroom/:slug/:code", "POST /api/public/showroom/:slug/:code/visit",
		"POST /api/public/lead/events", "PUT /api/public/lead/contact",
		"POST /api/public/lead/proposals",
		"GET /api/me/leads", "GET /api/me/leads/:id", "PATCH /api/me/leads/:id/stage",
		"POST /api/me/leads/:id/proposals", "GET /api/me/summary", "GET /api/me/activity", "GET /api/admin/activity",
		"GET /api/admin/leads", "GET /api/admin/leads/:id", "GET /api/admin/summary",
	} {
		assert.True(t, got[want], want)
	}
	for _, gone := range []string{"GET /api/public/lead/messages", "POST /api/public/lead/messages", "POST /api/me/leads/:id/messages"} {
		assert.False(t, got[gone], "o chat saiu (ADR-0015): "+gone)
	}
}
