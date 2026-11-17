package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neo360-api/internal/domain/broker"
	"neo360-api/internal/infra/api"
)

type BrokerHandler struct {
	svc broker.Service
	log broker.Logger
}

func NewBrokerHandler(svc broker.Service, log broker.Logger) *BrokerHandler {
	return &BrokerHandler{svc: svc, log: log}
}

// List godoc: GET /api/admin/brokers
func (h *BrokerHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	h.respond(c, http.StatusOK, list, err, "list brokers failed")
}

// Get godoc: GET /api/admin/brokers/:id
func (h *BrokerHandler) Get(c *gin.Context) {
	b, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, b, err, "get broker failed")
}

// Create godoc: POST /api/admin/brokers
func (h *BrokerHandler) Create(c *gin.Context) {
	var req broker.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	b, err := h.svc.Create(c.Request.Context(), req)
	h.respond(c, http.StatusCreated, b, err, "create broker failed")
}

// Update godoc: PUT /api/admin/brokers/:id
func (h *BrokerHandler) Update(c *gin.Context) {
	var req broker.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	b, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	h.respond(c, http.StatusOK, b, err, "update broker failed")
}

func (h *BrokerHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	Respond(c, h.log, status, data, err, msg)
}

// Respond traduz os erros do corretor em status HTTP. A conta (cadastro e perfil) usa a mesma tradução.
func Respond(c *gin.Context, log broker.Logger, status int, data any, err error, msg string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, broker.ErrNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, broker.ErrEmailTaken):
		api.RespondError(c, http.StatusConflict, err.Error())
	case errors.Is(err, broker.ErrNameRequired), errors.Is(err, broker.ErrNameTooLong), errors.Is(err, broker.ErrInvalidPhone),
		errors.Is(err, broker.ErrInvalidCreci), errors.Is(err, broker.ErrInvalidEmail), errors.Is(err, broker.ErrWeakPassword),
		errors.Is(err, broker.ErrAgencyNotFound):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}
