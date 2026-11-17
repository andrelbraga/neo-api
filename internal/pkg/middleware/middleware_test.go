package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neo360-api/internal/pkg/authz"
	"neo360-api/internal/pkg/security"
)

type fakeParser struct{}

func (fakeParser) Parse(token string) (security.Claims, error) {
	if token == "ok" {
		return security.Claims{UserID: "u1", Role: "corretor"}, nil
	}
	return security.Claims{}, errors.New("bad")
}

type fakeAccounts struct {
	role string
	err  error
}

func (f fakeAccounts) ActiveRole(context.Context, string) (string, error) { return f.role, f.err }

func init() { gin.SetMode(gin.TestMode) }

func serve(r *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthMiddleware(t *testing.T) {
	r := gin.New()
	r.GET("/x", AuthMiddleware(fakeParser{}, nil), func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("userID")+":"+c.GetString("role"))
	})

	cases := []struct {
		name   string
		header string
		status int
	}{
		{"sem header", "", http.StatusUnauthorized},
		{"esquema errado", "Basic ok", http.StatusUnauthorized},
		{"bearer vazio", "Bearer ", http.StatusUnauthorized},
		{"token inválido", "Bearer ruim", http.StatusUnauthorized},
		{"token válido", "Bearer ok", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := serve(r, http.MethodGet, "/x", map[string]string{"Authorization": tc.header})
			assert.Equal(t, tc.status, w.Code)
			if tc.status == http.StatusOK {
				assert.Equal(t, "u1:corretor", w.Body.String())
			} else {
				assert.Contains(t, w.Body.String(), `"error"`)
			}
		})
	}
}

// O papel e a situação valem pelo banco, não pelo que o token diz (RN-01.5).
func TestAuthMiddleware_Accounts(t *testing.T) {
	route := func(a Accounts) *gin.Engine {
		r := gin.New()
		r.GET("/x", AuthMiddleware(fakeParser{}, a), func(c *gin.Context) { c.String(http.StatusOK, c.GetString("role")) })
		return r
	}
	h := map[string]string{"Authorization": "Bearer ok"}

	w := serve(route(fakeAccounts{role: "admin"}), http.MethodGet, "/x", h)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "admin", w.Body.String(), "papel trocado vale sem novo login")

	w = serve(route(fakeAccounts{err: ErrAccountNotFound}), http.MethodGet, "/x", h)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "desativado perde o acesso na hora")
	assert.Contains(t, w.Body.String(), "sessão encerrada")

	w = serve(route(fakeAccounts{err: errors.New("db")}), http.MethodGet, "/x", h)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRequire(t *testing.T) {
	r := gin.New()
	setRole := func(role string) gin.HandlerFunc { return func(c *gin.Context) { c.Set("role", role) } }
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/admin", setRole("admin"), Require(authz.Admin), ok)
	r.GET("/corretor", setRole("corretor"), Require(authz.Admin), ok)
	r.GET("/sem-papel", Require(authz.BrokerArea), ok)

	assert.Equal(t, http.StatusOK, serve(r, http.MethodGet, "/admin", nil).Code)
	assert.Equal(t, http.StatusForbidden, serve(r, http.MethodGet, "/corretor", nil).Code)
	assert.Equal(t, http.StatusForbidden, serve(r, http.MethodGet, "/sem-papel", nil).Code)
}

func TestCORS(t *testing.T) {
	r := gin.New()
	r.Use(CORS([]string{"http://localhost:5173"}))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := serve(r, http.MethodGet, "/x", map[string]string{"Origin": "http://localhost:5173"})
	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "X-Visitor-Token")

	w = serve(r, http.MethodGet, "/x", map[string]string{"Origin": "https://evil.example"})
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))

	w = serve(r, http.MethodOptions, "/x", map[string]string{"Origin": "http://localhost:5173"})
	assert.Equal(t, http.StatusNoContent, w.Code)
}
