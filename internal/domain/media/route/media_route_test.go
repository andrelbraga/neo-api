package route

import (
	"io"
	"log/slog"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neo360-api/internal/config"
	"neo360-api/internal/infra/api"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := api.New(config.Configs{JWTSecret: "s", JWTExpiryHours: 1}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	Register(app, nil)
	got := map[string]bool{}
	for _, r := range app.Router().Routes() {
		got[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		"GET /api/public/media/brokers/:file", "POST /api/public/lead/photos",
		"PUT /api/me/photo", "DELETE /api/me/photo", "GET /api/me/leads/:id/photos/:photoID",
		"GET /api/admin/leads/:id/photos/:photoID",
	} {
		assert.True(t, got[want], want)
	}
}
