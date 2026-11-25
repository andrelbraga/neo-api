package handler

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
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
	invite string
	broker string
	devID  string
	data   string
	dryRun bool
	user   string
	filter lead.Filter
}

func (f *fake) Showroom(context.Context, string, string) (lead.Showroom, error) {
	return lead.Showroom{}, f.err
}
func (f *fake) Visit(_ context.Context, _, _, token, layout, invite string) (lead.Visit, error) {
	f.token, f.layout, f.invite = token, layout, invite
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
func (f fakeBroker) Create(_ context.Context, _ string, r lead.NewClientRequest) (lead.Lead, error) {
	f.devID = r.DevelopmentID
	return lead.Lead{}, f.err
}
func (f fakeBroker) Import(_ context.Context, _, devID string, data []byte, dryRun bool) (lead.ImportReport, error) {
	f.devID, f.data, f.dryRun = devID, string(data), dryRun
	return lead.ImportReport{}, f.err
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
	r.POST("/me/leads", h.CreateClient)
	r.POST("/me/leads/import", h.ImportClients)
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
	{http.MethodPost, "/me/leads", `{}`, http.StatusCreated},
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
	do(f, http.MethodPost, "/public/showroom/bv/FAB-0001/visit?convite=abc123", "")
	assert.Equal(t, "abc123", f.invite, "o código do convite vem da query da visita")
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
		{lead.ErrInvalidClientName, http.StatusBadRequest},
		{lead.ErrInvalidDevelopment, http.StatusBadRequest},
		{lead.ErrInvalidFile, http.StatusBadRequest},
		{lead.ErrTooManyRows, http.StatusBadRequest},
		{lead.ErrAlreadyYours, http.StatusConflict},
		{lead.ErrPhoneTaken, http.StatusConflict},
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

func upload(f *fake, fields map[string]string, file string, withFile bool) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewLeadHandler(f, fakeBroker{f}, fakeAdmin{f}, nopLog{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.POST("/me/leads/import", h.ImportClients)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	if withFile {
		w, _ := mw.CreateFormFile("file", "clientes.csv")
		_, _ = w.Write([]byte(file))
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/me/leads/import", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// RN-03.16: a planilha vem em multipart; sem dry_run=false, é só a conferência.
func TestImportClients(t *testing.T) {
	f := &fake{}
	w := upload(f, map[string]string{"development_id": "d1"}, "nome;telefone\nAna;54999990000", true)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "d1", f.devID)
	assert.Contains(t, f.data, "Ana;54999990000")
	assert.True(t, f.dryRun, "sem dry_run, só confere")

	f = &fake{}
	upload(f, map[string]string{"development_id": "d1", "dry_run": "false"}, "Ana;54999990000", true)
	assert.False(t, f.dryRun, "dry_run=false grava")

	assert.Equal(t, http.StatusBadRequest, upload(&fake{}, map[string]string{"development_id": "d1"}, "", false).Code, "sem arquivo")
	big := strings.Repeat("a", lead.MaxImportBytes+1)
	assert.Equal(t, http.StatusBadRequest, upload(&fake{}, map[string]string{"development_id": "d1"}, big, true).Code, "arquivo grande demais")
	assert.Equal(t, http.StatusConflict, upload(&fake{err: lead.ErrPhoneTaken}, nil, "x", true).Code)
}
