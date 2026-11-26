package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"neo360-api/internal/domain/media"
	"neo360-api/internal/infra/api"
	"neo360-api/internal/infra/storage"
)

// TokenHeader é o mesmo cabeçalho do token da visita do showroom (lead/handler).
const TokenHeader = "X-Visitor-Token"

type MediaHandler struct {
	svc media.Service
	log media.Logger
}

func NewMediaHandler(svc media.Service, log media.Logger) *MediaHandler {
	return &MediaHandler{svc: svc, log: log}
}

// SetBrokerPhoto godoc: PUT /api/me/photo (multipart, campo file)
func (h *MediaHandler) SetBrokerPhoto(c *gin.Context) {
	data, ok := h.file(c, media.MaxBrokerPhotoBytes)
	if !ok {
		return
	}
	p, err := h.svc.SetBrokerPhoto(c.Request.Context(), c.GetString("userID"), data)
	h.respond(c, http.StatusOK, p, err, "set broker photo failed")
}

// RemoveBrokerPhoto godoc: DELETE /api/me/photo
func (h *MediaHandler) RemoveBrokerPhoto(c *gin.Context) {
	err := h.svc.RemoveBrokerPhoto(c.Request.Context(), c.GetString("userID"))
	h.respond(c, http.StatusOK, media.BrokerPhoto{}, err, "remove broker photo failed")
}

// BrokerPhoto godoc: GET /api/public/media/brokers/:file
// O nome do arquivo muda a cada foto nova, então o navegador pode guardar para sempre.
func (h *MediaHandler) BrokerPhoto(c *gin.Context) {
	obj, err := h.svc.BrokerPhoto(c.Request.Context(), c.Param("file"))
	h.serve(c, obj, err, "public, max-age=31536000, immutable")
}

// UploadLeadPhoto godoc: POST /api/public/lead/photos (multipart, campo file; cabeçalho X-Visitor-Token)
func (h *MediaHandler) UploadLeadPhoto(c *gin.Context) {
	data, ok := h.file(c, media.MaxVehiclePhotoBytes)
	if !ok {
		return
	}
	p, err := h.svc.UploadLeadPhoto(c.Request.Context(), c.GetHeader(TokenHeader), data)
	h.respond(c, http.StatusCreated, p, err, "upload lead photo failed")
}

// MyLeadPhoto godoc: GET /api/me/leads/:id/photos/:photoID
func (h *MediaHandler) MyLeadPhoto(c *gin.Context) {
	obj, err := h.svc.LeadPhoto(c.Request.Context(), c.GetString("userID"), c.Param("id"), c.Param("photoID"))
	h.serve(c, obj, err, "private, max-age=3600")
}

// AdminLeadPhoto godoc: GET /api/admin/leads/:id/photos/:photoID
func (h *MediaHandler) AdminLeadPhoto(c *gin.Context) {
	obj, err := h.svc.LeadPhoto(c.Request.Context(), "", c.Param("id"), c.Param("photoID"))
	h.serve(c, obj, err, "private, max-age=3600")
}

// file lê o campo "file" do formulário, até o limite.
func (h *MediaHandler) file(c *gin.Context, limit int) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(limit)+64<<10)
	fh, err := c.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			h.respond(c, 0, nil, media.ErrTooLarge, "")
		} else {
			h.respond(c, 0, nil, media.ErrNotImage, "")
		}
		return nil, false
	}
	if fh.Size > int64(limit) {
		h.respond(c, 0, nil, media.ErrTooLarge, "")
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		h.respond(c, 0, nil, media.ErrNotImage, "")
		return nil, false
	}
	defer f.Close() //nolint:errcheck
	data, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		h.respond(c, 0, nil, media.ErrNotImage, "")
		return nil, false
	}
	return data, true
}

func (h *MediaHandler) serve(c *gin.Context, obj storage.Object, err error, cache string) {
	if err != nil {
		h.respond(c, 0, nil, err, "read photo failed")
		return
	}
	defer obj.Body.Close() //nolint:errcheck
	c.DataFromReader(http.StatusOK, obj.Size, obj.ContentType, obj.Body, map[string]string{
		"Cache-Control":          cache,
		"X-Content-Type-Options": "nosniff",
	})
}

func (h *MediaHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, media.ErrNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, media.ErrInvalidToken):
		api.RespondError(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, media.ErrBrokerNotFound):
		api.RespondError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, media.ErrTooLarge):
		api.RespondError(c, http.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, media.ErrNotImage), errors.Is(err, media.ErrTooMany):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, media.ErrUnavailable):
		api.RespondError(c, http.StatusServiceUnavailable, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}
