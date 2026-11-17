package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neo360-api/internal/domain/organization"
	"neo360-api/internal/infra/api"
)

type OrganizationHandler struct {
	svc organization.Service
	log organization.Logger
}

func NewOrganizationHandler(svc organization.Service, log organization.Logger) *OrganizationHandler {
	return &OrganizationHandler{svc: svc, log: log}
}

// List godoc: GET /api/admin/organizations?kind=incorporadora|imobiliaria
func (h *OrganizationHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), c.Query("kind"))
	h.respond(c, http.StatusOK, list, err, "list organizations failed")
}

// Get godoc: GET /api/admin/organizations/:id
func (h *OrganizationHandler) Get(c *gin.Context) {
	o, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, o, err, "get organization failed")
}

// Create godoc: POST /api/admin/organizations
func (h *OrganizationHandler) Create(c *gin.Context) {
	var req organization.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	o, err := h.svc.Create(c.Request.Context(), req)
	h.respond(c, http.StatusCreated, o, err, "create organization failed")
}

// Update godoc: PUT /api/admin/organizations/:id
func (h *OrganizationHandler) Update(c *gin.Context) {
	var req organization.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	o, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	h.respond(c, http.StatusOK, o, err, "update organization failed")
}

func (h *OrganizationHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, organization.ErrNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, organization.ErrInvalidKind), errors.Is(err, organization.ErrNameRequired),
		errors.Is(err, organization.ErrFieldTooLong), errors.Is(err, organization.ErrInvalidEmail),
		errors.Is(err, organization.ErrInvalidLogo):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}
