package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neo360-api/internal/domain/account"
	"neo360-api/internal/domain/broker"
)

type fakeService struct {
	err  error
	user string
}

func (f *fakeService) Login(context.Context, account.LoginRequest) (account.Session, error) {
	return account.Session{Token: "t"}, f.err
}
func (f *fakeService) Signup(context.Context, broker.CreateRequest) (account.Session, error) {
	return account.Session{Token: "t"}, f.err
}
func (f *fakeService) Me(_ context.Context, id string) (account.User, error) {
	f.user = id
	return account.User{ID: id}, f.err
}
func (f *fakeService) UpdateProfile(_ context.Context, id string, _ broker.Profile) (account.User, error) {
	f.user = id
	return account.User{ID: id}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc account.Service, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewAccountHandler(svc, nopLog{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.POST("/auth/login", h.Login)
	r.POST("/auth/signup", h.Signup)
	r.GET("/auth/me", h.Me)
	r.PUT("/me/profile", h.UpdateProfile)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

var routes = []struct {
	method, path, body string
	status             int
}{
	{http.MethodPost, "/auth/login", `{}`, http.StatusOK},
	{http.MethodPost, "/auth/signup", `{}`, http.StatusCreated},
	{http.MethodGet, "/auth/me", "", http.StatusOK},
	{http.MethodPut, "/me/profile", `{}`, http.StatusOK},
}

func TestAccountHandler(t *testing.T) {
	svc := &fakeService{}
	for _, r := range routes {
		assert.Equal(t, r.status, do(svc, r.method, r.path, r.body).Code, r.path)
	}
	assert.Equal(t, "u1", svc.user, "o usuário vem do token")
	for _, p := range []string{"/auth/login", "/auth/signup"} {
		assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, http.MethodPost, p, `{`).Code, p)
	}
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, http.MethodPut, "/me/profile", `{`).Code)

	cases := []struct {
		err    error
		status int
	}{
		{account.ErrInvalidCredentials, http.StatusUnauthorized},
		{account.ErrNotFound, http.StatusNotFound},
		{broker.ErrEmailTaken, http.StatusConflict},
		{broker.ErrInvalidPhone, http.StatusBadRequest},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		for _, r := range routes {
			assert.Equal(t, tc.status, do(&fakeService{err: tc.err}, r.method, r.path, r.body).Code, tc.err.Error()+" "+r.path)
		}
	}
}
