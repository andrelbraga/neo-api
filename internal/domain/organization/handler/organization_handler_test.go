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

	"neo360-api/internal/domain/organization"
)

type fakeService struct {
	err  error
	kind string
}

func (f *fakeService) List(_ context.Context, kind string) ([]organization.Organization, error) {
	f.kind = kind
	return nil, f.err
}
func (f *fakeService) Get(context.Context, string) (organization.Organization, error) {
	return organization.Organization{}, f.err
}
func (f *fakeService) Create(context.Context, organization.Request) (organization.Organization, error) {
	return organization.Organization{}, f.err
}
func (f *fakeService) Update(context.Context, string, organization.Request) (organization.Organization, error) {
	return organization.Organization{}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc organization.Service, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewOrganizationHandler(svc, nopLog{})
	r := gin.New()
	r.GET("/organizations", h.List)
	r.GET("/organizations/:id", h.Get)
	r.POST("/organizations", h.Create)
	r.PUT("/organizations/:id", h.Update)
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
	{http.MethodGet, "/organizations?kind=imobiliaria", "", http.StatusOK},
	{http.MethodGet, "/organizations/o1", "", http.StatusOK},
	{http.MethodPost, "/organizations", `{}`, http.StatusCreated},
	{http.MethodPut, "/organizations/o1", `{}`, http.StatusOK},
}

func TestOrganizationHandler(t *testing.T) {
	svc := &fakeService{}
	for _, r := range routes {
		assert.Equal(t, r.status, do(svc, r.method, r.path, r.body).Code, r.path)
	}
	assert.Equal(t, "imobiliaria", svc.kind)
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, http.MethodPost, "/organizations", `{`).Code)
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, http.MethodPut, "/organizations/o1", `{`).Code)

	cases := []struct {
		err    error
		status int
	}{
		{organization.ErrNotFound, http.StatusNotFound},
		{organization.ErrInvalidKind, http.StatusBadRequest},
		{organization.ErrNameRequired, http.StatusBadRequest},
		{organization.ErrFieldTooLong, http.StatusBadRequest},
		{organization.ErrInvalidEmail, http.StatusBadRequest},
		{organization.ErrInvalidLogo, http.StatusBadRequest},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		for _, r := range routes {
			assert.Equal(t, tc.status, do(&fakeService{err: tc.err}, r.method, r.path, r.body).Code, tc.err.Error()+" "+r.path)
		}
	}
}
