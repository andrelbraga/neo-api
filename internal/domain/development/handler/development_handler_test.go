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

	"neo360-api/internal/domain/development"
)

type fakeService struct {
	err       error
	published []bool
}

func (f *fakeService) List(_ context.Context, onlyPublished bool) ([]development.Development, error) {
	f.published = append(f.published, onlyPublished)
	return nil, f.err
}
func (f *fakeService) Get(context.Context, string) (development.Development, error) {
	return development.Development{}, f.err
}
func (f *fakeService) GetPublishedBySlug(context.Context, string) (development.Development, error) {
	return development.Development{}, f.err
}
func (f *fakeService) Create(context.Context, development.Request) (development.Development, error) {
	return development.Development{}, f.err
}
func (f *fakeService) Update(context.Context, string, development.Request) (development.Development, error) {
	return development.Development{}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc development.Service, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewDevelopmentHandler(svc, nopLog{})
	r := gin.New()
	r.GET("/admin/developments", h.List)
	r.GET("/me/developments", h.ListPublished)
	r.GET("/admin/developments/:id", h.Get)
	r.POST("/admin/developments", h.Create)
	r.PUT("/admin/developments/:id", h.Update)
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
	{http.MethodGet, "/admin/developments", "", http.StatusOK},
	{http.MethodGet, "/me/developments", "", http.StatusOK},
	{http.MethodGet, "/admin/developments/d1", "", http.StatusOK},
	{http.MethodPost, "/admin/developments", `{}`, http.StatusCreated},
	{http.MethodPut, "/admin/developments/d1", `{}`, http.StatusOK},
}

func TestDevelopmentHandler(t *testing.T) {
	svc := &fakeService{}
	for _, r := range routes {
		assert.Equal(t, r.status, do(svc, r.method, r.path, r.body).Code, r.path)
	}
	assert.Equal(t, []bool{false, true}, svc.published, "admin vê todos; corretor só os publicados")
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, http.MethodPost, "/admin/developments", `{`).Code)
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, http.MethodPut, "/admin/developments/d1", `{`).Code)

	cases := []struct {
		err    error
		status int
	}{
		{development.ErrNotFound, http.StatusNotFound},
		{development.ErrSlugTaken, http.StatusConflict},
		{development.ErrNameRequired, http.StatusBadRequest},
		{development.ErrInvalidSlug, http.StatusBadRequest},
		{development.ErrInvalidStatus, http.StatusBadRequest},
		{development.ErrInvalidState, http.StatusBadRequest},
		{development.ErrInvalidURL, http.StatusBadRequest},
		{development.ErrInvalidHighlight, http.StatusBadRequest},
		{development.ErrInvalidLayout, http.StatusBadRequest},
		{development.ErrTooManyItems, http.StatusBadRequest},
		{development.ErrFieldTooLong, http.StatusBadRequest},
		{development.ErrDeveloperNotFound, http.StatusBadRequest},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		for _, r := range routes {
			assert.Equal(t, tc.status, do(&fakeService{err: tc.err}, r.method, r.path, r.body).Code, tc.err.Error()+" "+r.path)
		}
	}
}
