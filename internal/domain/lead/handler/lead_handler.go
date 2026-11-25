package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"neo360-api/internal/domain/lead"
	"neo360-api/internal/infra/api"
)

// TokenHeader leva o token da visita do cliente no showroom.
const TokenHeader = "X-Visitor-Token"

type LeadHandler struct {
	public lead.PublicService
	broker lead.BrokerService
	admin  lead.AdminService
	log    lead.Logger
}

func NewLeadHandler(public lead.PublicService, broker lead.BrokerService, admin lead.AdminService, log lead.Logger) *LeadHandler {
	return &LeadHandler{public: public, broker: broker, admin: admin, log: log}
}

// ─── Showroom (público) ──────────────────────────────────────────────────────

// Showroom godoc: GET /api/public/showroom/:slug/:code
func (h *LeadHandler) Showroom(c *gin.Context) {
	room, err := h.public.Showroom(c.Request.Context(), c.Param("slug"), c.Param("code"))
	h.respond(c, http.StatusOK, room, err, "showroom failed")
}

// Visit godoc: POST /api/public/showroom/:slug/:code/visit?layout=a|b&convite=
func (h *LeadHandler) Visit(c *gin.Context) {
	v, err := h.public.Visit(c.Request.Context(), c.Param("slug"), c.Param("code"), c.GetHeader(TokenHeader), c.Query("layout"), c.Query("convite"))
	h.respond(c, http.StatusOK, v, err, "visit failed")
}

// Track godoc: POST /api/public/lead/events
func (h *LeadHandler) Track(c *gin.Context) {
	var req lead.EventRequest
	if !bind(c, &req) {
		return
	}
	err := h.public.Track(c.Request.Context(), c.GetHeader(TokenHeader), req)
	h.respond(c, http.StatusCreated, gin.H{"ok": true}, err, "track failed")
}

// Contact godoc: PUT /api/public/lead/contact
func (h *LeadHandler) Contact(c *gin.Context) {
	var req lead.ContactRequest
	if !bind(c, &req) {
		return
	}
	v, err := h.public.Contact(c.Request.Context(), c.GetHeader(TokenHeader), req)
	h.respond(c, http.StatusOK, v, err, "contact failed")
}

// Propose godoc: POST /api/public/lead/proposals
func (h *LeadHandler) Propose(c *gin.Context) {
	var req lead.ProposalRequest
	if !bind(c, &req) {
		return
	}
	p, err := h.public.Propose(c.Request.Context(), c.GetHeader(TokenHeader), req)
	h.respond(c, http.StatusCreated, p, err, "propose failed")
}

// ─── Corretor ────────────────────────────────────────────────────────────────

// MyLeads godoc: GET /api/me/leads?stage=&development=
func (h *LeadHandler) MyLeads(c *gin.Context) {
	list, err := h.broker.List(c.Request.Context(), c.GetString("userID"), filter(c))
	h.respond(c, http.StatusOK, list, err, "my leads failed")
}

// MyLead godoc: GET /api/me/leads/:id
func (h *LeadHandler) MyLead(c *gin.Context) {
	d, err := h.broker.Get(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	h.respond(c, http.StatusOK, d, err, "my lead failed")
}

// SetStage godoc: PATCH /api/me/leads/:id/stage
func (h *LeadHandler) SetStage(c *gin.Context) {
	var req lead.StageRequest
	if !bind(c, &req) {
		return
	}
	l, err := h.broker.SetStage(c.Request.Context(), c.GetString("userID"), c.Param("id"), req)
	h.respond(c, http.StatusOK, l, err, "set stage failed")
}

// Adjust godoc: POST /api/me/leads/:id/proposals
func (h *LeadHandler) Adjust(c *gin.Context) {
	var req lead.AdjustRequest
	if !bind(c, &req) {
		return
	}
	p, err := h.broker.Adjust(c.Request.Context(), c.GetString("userID"), c.Param("id"), req)
	h.respond(c, http.StatusCreated, p, err, "adjust proposal failed")
}

// MySummary godoc: GET /api/me/summary
func (h *LeadHandler) MySummary(c *gin.Context) {
	s, err := h.broker.Summary(c.Request.Context(), c.GetString("userID"))
	h.respond(c, http.StatusOK, s, err, "my summary failed")
}

// MyActivity godoc: GET /api/me/activity
func (h *LeadHandler) MyActivity(c *gin.Context) {
	list, err := h.broker.Activity(c.Request.Context(), c.GetString("userID"))
	h.respond(c, http.StatusOK, list, err, "my activity failed")
}

// CreateClient godoc: POST /api/me/leads
func (h *LeadHandler) CreateClient(c *gin.Context) {
	var req lead.NewClientRequest
	if !bind(c, &req) {
		return
	}
	l, err := h.broker.Create(c.Request.Context(), c.GetString("userID"), req)
	h.respond(c, http.StatusCreated, l, err, "create client failed")
}

// ImportClients godoc: POST /api/me/leads/import (multipart: file, development_id, dry_run=true|false)
func (h *LeadHandler) ImportClients(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, lead.MaxImportBytes+64<<10)
	fh, err := c.FormFile("file")
	if err != nil {
		h.respond(c, 0, nil, lead.ErrInvalidFile, "")
		return
	}
	if fh.Size > lead.MaxImportBytes {
		h.respond(c, 0, nil, lead.ErrTooManyRows, "")
		return
	}
	f, err := fh.Open()
	if err != nil {
		h.respond(c, 0, nil, lead.ErrInvalidFile, "")
		return
	}
	defer f.Close() //nolint:errcheck
	data, err := io.ReadAll(io.LimitReader(f, lead.MaxImportBytes+1))
	if err != nil {
		h.respond(c, 0, nil, lead.ErrInvalidFile, "")
		return
	}
	report, err := h.broker.Import(c.Request.Context(), c.GetString("userID"), c.PostForm("development_id"), data, c.PostForm("dry_run") != "false")
	h.respond(c, http.StatusOK, report, err, "import clients failed")
}

// UpdateClient godoc: PATCH /api/me/leads/:id
func (h *LeadHandler) UpdateClient(c *gin.Context) {
	var req lead.UpdateRequest
	if !bind(c, &req) {
		return
	}
	l, err := h.broker.Update(c.Request.Context(), c.GetString("userID"), c.Param("id"), req)
	h.respond(c, http.StatusOK, l, err, "update client failed")
}

// MessageSent godoc: POST /api/me/leads/:id/message-sent (fila de envio, ADR-0028)
func (h *LeadHandler) MessageSent(c *gin.Context) {
	var req lead.MessageSentRequest
	if !bind(c, &req) {
		return
	}
	l, err := h.broker.MessageSent(c.Request.Context(), c.GetString("userID"), c.Param("id"), req)
	h.respond(c, http.StatusOK, l, err, "message sent failed")
}

// ArchiveClient godoc: DELETE /api/me/leads/:id (arquiva; nada é apagado)
func (h *LeadHandler) ArchiveClient(c *gin.Context) {
	l, err := h.broker.Archive(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	h.respond(c, http.StatusOK, l, err, "archive client failed")
}

// RestoreClient godoc: POST /api/me/leads/:id/restore
func (h *LeadHandler) RestoreClient(c *gin.Context) {
	l, err := h.broker.Restore(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	h.respond(c, http.StatusOK, l, err, "restore client failed")
}

// Templates godoc: GET /api/me/message-templates
func (h *LeadHandler) Templates(c *gin.Context) {
	list, err := h.broker.Templates(c.Request.Context(), c.GetString("userID"))
	h.respond(c, http.StatusOK, list, err, "templates failed")
}

// SaveTemplate godoc: PUT /api/me/message-templates/:profile/:tone
func (h *LeadHandler) SaveTemplate(c *gin.Context) {
	var req lead.TemplateRequest
	if !bind(c, &req) {
		return
	}
	t, err := h.broker.SaveTemplate(c.Request.Context(), c.GetString("userID"), c.Param("profile"), c.Param("tone"), req)
	h.respond(c, http.StatusOK, t, err, "save template failed")
}

// ResetTemplate godoc: DELETE /api/me/message-templates/:profile/:tone (volta ao padrão)
func (h *LeadHandler) ResetTemplate(c *gin.Context) {
	t, err := h.broker.ResetTemplate(c.Request.Context(), c.GetString("userID"), c.Param("profile"), c.Param("tone"))
	h.respond(c, http.StatusOK, t, err, "reset template failed")
}

// ─── Admin ───────────────────────────────────────────────────────────────────

// AdminLeads godoc: GET /api/admin/leads?stage=&development=&broker=
func (h *LeadHandler) AdminLeads(c *gin.Context) {
	f := filter(c)
	f.BrokerID = c.Query("broker")
	list, err := h.admin.List(c.Request.Context(), f)
	h.respond(c, http.StatusOK, list, err, "admin leads failed")
}

// AdminLead godoc: GET /api/admin/leads/:id
func (h *LeadHandler) AdminLead(c *gin.Context) {
	d, err := h.admin.Get(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, d, err, "admin lead failed")
}

// AdminSummary godoc: GET /api/admin/summary?broker=
func (h *LeadHandler) AdminSummary(c *gin.Context) {
	s, err := h.admin.Summary(c.Request.Context(), c.Query("broker"))
	h.respond(c, http.StatusOK, s, err, "admin summary failed")
}

// AdminActivity godoc: GET /api/admin/activity?broker=
func (h *LeadHandler) AdminActivity(c *gin.Context) {
	list, err := h.admin.Activity(c.Request.Context(), c.Query("broker"))
	h.respond(c, http.StatusOK, list, err, "admin activity failed")
}

func filter(c *gin.Context) lead.Filter {
	return lead.Filter{Stage: c.Query("stage"), DevelopmentID: c.Query("development"), Archived: c.Query("arquivados") == "1"}
}

func bind(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return false
	}
	return true
}

func (h *LeadHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, lead.ErrNotFound), errors.Is(err, lead.ErrShowroomNotFound), errors.Is(err, lead.ErrTemplateKey):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, lead.ErrInvalidToken):
		api.RespondError(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, lead.ErrNotBroker):
		api.RespondError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, lead.ErrAlreadyYours), errors.Is(err, lead.ErrPhoneTaken):
		api.RespondError(c, http.StatusConflict, err.Error())
	case errors.Is(err, lead.ErrInvalidClientName), errors.Is(err, lead.ErrInvalidDevelopment), errors.Is(err, lead.ErrInvalidFile),
		errors.Is(err, lead.ErrTooManyRows), errors.Is(err, lead.ErrInvalidProfile), errors.Is(err, lead.ErrInvalidTemplate),
		errors.Is(err, lead.ErrInvalidMessage), errors.Is(err, lead.ErrNoPhone):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, lead.ErrInvalidEvent), errors.Is(err, lead.ErrInvalidName), errors.Is(err, lead.ErrInvalidPhone),
		errors.Is(err, lead.ErrInvalidProposal), errors.Is(err, lead.ErrInvalidSale), errors.Is(err, lead.ErrInvalidStage):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}
