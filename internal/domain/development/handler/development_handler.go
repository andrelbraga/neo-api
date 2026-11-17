package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neo360-api/internal/domain/development"
	"neo360-api/internal/infra/api"
)

type DevelopmentHandler struct {
	svc development.Service
	log development.Logger
}

func NewDevelopmentHandler(svc development.Service, log development.Logger) *DevelopmentHandler {
	return &DevelopmentHandler{svc: svc, log: log}
}

// List godoc: GET /api/admin/developments (todos)
func (h *DevelopmentHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), false)
	h.respond(c, http.StatusOK, list, err, "list developments failed")
}

// ListPublished godoc: GET /api/me/developments (o que o corretor pode ofertar, RN-02.3)
func (h *DevelopmentHandler) ListPublished(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), true)
	h.respond(c, http.StatusOK, list, err, "list published developments failed")
}

// Get godoc: GET /api/admin/developments/:id
func (h *DevelopmentHandler) Get(c *gin.Context) {
	d, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, d, err, "get development failed")
}

// Create godoc: POST /api/admin/developments
func (h *DevelopmentHandler) Create(c *gin.Context) {
	var req development.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	d, err := h.svc.Create(c.Request.Context(), req)
	h.respond(c, http.StatusCreated, d, err, "create development failed")
}

// Update godoc: PUT /api/admin/developments/:id
func (h *DevelopmentHandler) Update(c *gin.Context) {
	var req development.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	d, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	h.respond(c, http.StatusOK, d, err, "update development failed")
}

func (h *DevelopmentHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, development.ErrNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, development.ErrSlugTaken):
		api.RespondError(c, http.StatusConflict, err.Error())
	case errors.Is(err, development.ErrNameRequired), errors.Is(err, development.ErrInvalidSlug),
		errors.Is(err, development.ErrInvalidStatus), errors.Is(err, development.ErrInvalidState),
		errors.Is(err, development.ErrInvalidURL), errors.Is(err, development.ErrInvalidHighlight),
		errors.Is(err, development.ErrTooManyItems), errors.Is(err, development.ErrFieldTooLong),
		errors.Is(err, development.ErrDeveloperNotFound):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}
