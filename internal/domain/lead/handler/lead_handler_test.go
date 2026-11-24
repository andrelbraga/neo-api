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

	"neo360-api/internal/domain/lead"
)

// fake atende os três serviços e guarda o que recebeu.
type fake struct {
	err    error
	token  string
	layout string
	broker string
	user   string
	filter lead.Filter
}

func (f *fake) Showroom(context.Context, string, string) (lead.Showroom, error) {
	return lead.Showroom{}, f.err
}
func (f *fake) Visit(_ context.Context, _, _, token, layout string) (lead.Visit, error) {
	f.token, f.layout = token, layout
	return lead.Visit{}, f.err
}
func (f *fake) Track(_ context.Context, token string, _ lead.EventRequest) error {
	f.token = token
	return f.err
}
func (f *fake) Contact(context.Context, string, lead.ContactRequest) (lead.Visit, error) {
	return lead.Visit{}, f.err
}
func (f *fake) Propose(_ context.Context, token string, _ lead.ProposalRequest) (lead.Proposal, error) {
	f.token = token
	return lead.Proposal{}, f.err
}

type fakeBroker struct{ *fake }

func (f fakeBroker) List(_ context.Context, user string, fl lead.Filter) ([]lead.Lead, error) {
	f.user, f.filter = user, fl
	return nil, f.err
}
func (f fakeBroker) Get(context.Context, string, string) (lead.Detail, error) {
	return lead.Detail{}, f.err
}
func (f fakeBroker) SetStage(context.Context, string, string, lead.StageRequest) (lead.Lead, error) {
	return lead.Lead{}, f.err
}
func (f fakeBroker) Adjust(context.Context, string, string, lead.AdjustRequest) (lead.Proposal, error) {
	return lead.Proposal{}, f.err
}
func (f fakeBroker) Summary(context.Context, string) (lead.Summary, error) {
	return lead.Summary{}, f.err
}
func (f fakeBroker) Activity(context.Context, string) ([]lead.Activity, error) { return nil, f.err }

type fakeAdmin struct{ *fake }

func (f fakeAdmin) List(_ context.Context, fl lead.Filter) ([]lead.Lead, error) {
	f.filter = fl
	return nil, f.err
}
func (f fakeAdmin) Get(context.Context, string) (lead.Detail, error) { return lead.Detail{}, f.err }
func (f fakeAdmin) Summary(_ context.Context, broker string) (lead.Summary, error) {
	f.broker = broker
	return lead.Summary{}, f.err
}
func (f fakeAdmin) Activity(_ context.Context, broker string) ([]lead.Activity, error) {
	f.broker = broker
	return nil, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(f *fake, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewLeadHandler(f, fakeBroker{f}, fakeAdmin{f}, nopLog{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.GET("/public/showroom/:slug/:code", h.Showroom)
	r.POST("/public/showroom/:slug/:code/visit", h.Visit)
	r.POST("/public/lead/events", h.Track)
	r.PUT("/public/lead/contact", h.Contact)
	r.POST("/public/lead/proposals", h.Propose)
	r.GET("/me/leads", h.MyLeads)
	r.GET("/me/leads/:id", h.MyLead)
	r.PATCH("/me/leads/:id/stage", h.SetStage)
	r.POST("/me/leads/:id/proposals", h.Adjust)
	r.GET("/me/summary", h.MySummary)
	r.GET("/me/activity", h.MyActivity)
	r.GET("/admin/leads", h.AdminLeads)
	r.GET("/admin/leads/:id", h.AdminLead)
	r.GET("/admin/summary", h.AdminSummary)
	r.GET("/admin/activity", h.AdminActivity)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(TokenHeader, "tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

var routes = []struct {
	method, path, body string
	status             int
}{
	{http.MethodGet, "/public/showroom/bv/FAB-0001", "", http.StatusOK},
	{http.MethodPost, "/public/showroom/bv/FAB-0001/visit", "", http.StatusOK},
	{http.MethodPost, "/public/lead/events", `{}`, http.StatusCreated},
	{http.MethodPut, "/public/lead/contact", `{}`, http.StatusOK},
	{http.MethodPost, "/public/lead/proposals", `{}`, http.StatusCreated},
	{http.MethodGet, "/me/leads?stage=novo&development=d1", "", http.StatusOK},
	{http.MethodGet, "/me/leads/l1", "", http.StatusOK},
	{http.MethodPatch, "/me/leads/l1/stage", `{}`, http.StatusOK},
	{http.MethodPost, "/me/leads/l1/proposals", `{}`, http.StatusCreated},
	{http.MethodGet, "/me/summary", "", http.StatusOK},
	{http.MethodGet, "/me/activity", "", http.StatusOK},
	{http.MethodGet, "/admin/leads?broker=b1", "", http.StatusOK},
	{http.MethodGet, "/admin/leads/l1", "", http.StatusOK},
	{http.MethodGet, "/admin/summary", "", http.StatusOK},
	{http.MethodGet, "/admin/activity", "", http.StatusOK},
}

func TestLeadHandler(t *testing.T) {
	f := &fake{}
	for _, r := range routes {
		assert.Equal(t, r.status, do(f, r.method, r.path, r.body).Code, r.path)
	}

	f = &fake{}
	do(f, http.MethodPost, "/public/lead/events", `{}`)
	assert.Equal(t, "tok", f.token, "o token vem do cabeçalho")
	f = &fake{}
	do(f, http.MethodPost, "/public/showroom/bv/FAB-0001/visit?layout=b", "")
	assert.Equal(t, "b", f.layout, "o layout vem da query da visita")
	f = &fake{}
	do(f, http.MethodPost, "/public/lead/proposals", `{}`)
	assert.Equal(t, "tok", f.token, "a proposta também vem com o token da visita")

	for _, r := range routes {
		if r.body != "" {
			assert.Equal(t, http.StatusBadRequest, do(&fake{}, r.method, r.path, `{`).Code, r.path)
		}
	}

	cases := []struct {
		err    error
		status int
	}{
		{lead.ErrNotFound, http.StatusNotFound},
		{lead.ErrShowroomNotFound, http.StatusNotFound},
		{lead.ErrInvalidToken, http.StatusUnauthorized},
		{lead.ErrNotBroker, http.StatusForbidden},
		{lead.ErrInvalidEvent, http.StatusBadRequest},
		{lead.ErrInvalidName, http.StatusBadRequest},
		{lead.ErrInvalidPhone, http.StatusBadRequest},
		{lead.ErrInvalidProposal, http.StatusBadRequest},
		{lead.ErrInvalidSale, http.StatusBadRequest},
		{lead.ErrInvalidStage, http.StatusBadRequest},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		for _, r := range routes {
			assert.Equal(t, tc.status, do(&fake{err: tc.err}, r.method, r.path, r.body).Code, tc.err.Error()+" "+r.path)
		}
	}
}

func TestFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var got lead.Filter
	r := gin.New()
	r.GET("/x", func(c *gin.Context) { got = filter(c) })
	req := httptest.NewRequest(http.MethodGet, "/x?stage=proposta&development=d1&broker=b9", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, lead.Filter{Stage: "proposta", DevelopmentID: "d1"}, got, "o filtro de corretor só entra no admin")
}

// RN-01.10: o "ver como" do admin passa o corretor ao resumo e ao feed.
func TestAdminSeeAs(t *testing.T) {
	for _, path := range []string{"/admin/summary?broker=b7", "/admin/activity?broker=b7"} {
		f := &fake{}
		assert.Equal(t, http.StatusOK, do(f, http.MethodGet, path, "").Code)
		assert.Equal(t, "b7", f.broker, path)
	}
}
