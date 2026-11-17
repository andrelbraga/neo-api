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

	"neo360-api/internal/domain/broker"
)

type fakeService struct{ err error }

func (f fakeService) Create(context.Context, broker.CreateRequest) (broker.Broker, error) {
	return broker.Broker{ID: "b1"}, f.err
}
func (f fakeService) List(context.Context) ([]broker.Broker, error) { return nil, f.err }
func (f fakeService) Get(_ context.Context, id string) (broker.Broker, error) {
	return broker.Broker{ID: id}, f.err
}
func (f fakeService) GetByUser(context.Context, string) (broker.Broker, error) {
	return broker.Broker{}, f.err
}
func (f fakeService) Update(_ context.Context, id string, _ broker.UpdateRequest) (broker.Broker, error) {
	return broker.Broker{ID: id}, f.err
}
func (f fakeService) UpdateProfile(context.Context, string, broker.Profile) (broker.Broker, error) {
	return broker.Broker{}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc broker.Service, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewBrokerHandler(svc, nopLog{})
	r := gin.New()
	r.GET("/brokers", h.List)
	r.GET("/brokers/:id", h.Get)
	r.POST("/brokers", h.Create)
	r.PUT("/brokers/:id", h.Update)
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
	{http.MethodGet, "/brokers", "", http.StatusOK},
	{http.MethodGet, "/brokers/b1", "", http.StatusOK},
	{http.MethodPost, "/brokers", `{}`, http.StatusCreated},
	{http.MethodPut, "/brokers/b1", `{}`, http.StatusOK},
}

func TestBrokerHandler(t *testing.T) {
	for _, r := range routes {
		assert.Equal(t, r.status, do(fakeService{}, r.method, r.path, r.body).Code, r.path)
	}
	assert.Equal(t, http.StatusBadRequest, do(fakeService{}, http.MethodPost, "/brokers", `{`).Code)
	assert.Equal(t, http.StatusBadRequest, do(fakeService{}, http.MethodPut, "/brokers/b1", `{`).Code)

	cases := []struct {
		err    error
		status int
	}{
		{broker.ErrNotFound, http.StatusNotFound},
		{broker.ErrEmailTaken, http.StatusConflict},
		{broker.ErrNameRequired, http.StatusBadRequest},
		{broker.ErrNameTooLong, http.StatusBadRequest},
		{broker.ErrInvalidPhone, http.StatusBadRequest},
		{broker.ErrInvalidCreci, http.StatusBadRequest},
		{broker.ErrInvalidEmail, http.StatusBadRequest},
		{broker.ErrWeakPassword, http.StatusBadRequest},
		{broker.ErrAgencyNotFound, http.StatusBadRequest},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		for _, r := range routes {
			assert.Equal(t, tc.status, do(fakeService{err: tc.err}, r.method, r.path, r.body).Code, tc.err.Error()+" "+r.path)
		}
	}
}
