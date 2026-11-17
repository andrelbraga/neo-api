package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neo360-api/internal/domain/account"
	"neo360-api/internal/domain/broker"
	brokerHandler "neo360-api/internal/domain/broker/handler"
	"neo360-api/internal/infra/api"
)

type AccountHandler struct {
	svc account.Service
	log account.Logger
}

func NewAccountHandler(svc account.Service, log account.Logger) *AccountHandler {
	return &AccountHandler{svc: svc, log: log}
}

// Login godoc: POST /api/auth/login
func (h *AccountHandler) Login(c *gin.Context) {
	var req account.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	sess, err := h.svc.Login(c.Request.Context(), req)
	h.respond(c, http.StatusOK, sess, err, "login failed")
}

// Signup godoc: POST /api/auth/signup
func (h *AccountHandler) Signup(c *gin.Context) {
	var req broker.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	sess, err := h.svc.Signup(c.Request.Context(), req)
	h.respond(c, http.StatusCreated, sess, err, "signup failed")
}

// Me godoc: GET /api/auth/me
func (h *AccountHandler) Me(c *gin.Context) {
	u, err := h.svc.Me(c.Request.Context(), c.GetString("userID"))
	h.respond(c, http.StatusOK, u, err, "me failed")
}

// UpdateProfile godoc: PUT /api/me/profile
func (h *AccountHandler) UpdateProfile(c *gin.Context) {
	var req broker.Profile
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	u, err := h.svc.UpdateProfile(c.Request.Context(), c.GetString("userID"), req)
	h.respond(c, http.StatusOK, u, err, "update profile failed")
}

// ChangePassword godoc: PUT /api/auth/password
func (h *AccountHandler) ChangePassword(c *gin.Context) {
	var req account.PasswordChange
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	u, err := h.svc.ChangePassword(c.Request.Context(), c.GetString("userID"), req)
	h.respond(c, http.StatusOK, u, err, "change password failed")
}

// Accounts godoc: GET /api/admin/users
func (h *AccountHandler) Accounts(c *gin.Context) {
	list, err := h.svc.ListAccounts(c.Request.Context())
	h.respond(c, http.StatusOK, list, err, "list accounts failed")
}

// ResetPassword godoc: POST /api/admin/users/:id/reset-password
func (h *AccountHandler) ResetPassword(c *gin.Context) {
	temp, err := h.svc.ResetPassword(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, temp, err, "reset password failed")
}

func (h *AccountHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	switch {
	case errors.Is(err, account.ErrInvalidCredentials):
		api.RespondError(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, account.ErrNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, account.ErrWeakPassword), errors.Is(err, account.ErrSamePassword), errors.Is(err, account.ErrWrongPassword):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		brokerHandler.Respond(c, h.log, status, data, err, msg)
	}
}
