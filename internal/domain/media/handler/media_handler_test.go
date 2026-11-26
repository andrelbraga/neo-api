package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neo360-api/internal/domain/media"
	"neo360-api/internal/infra/storage"
)

type fakeService struct {
	err       error
	gotData   []byte
	gotToken  string
	gotUser   string
	gotFile   string
	gotLead   string
	gotPhoto  string
	removedBy string
}

func (f *fakeService) SetBrokerPhoto(_ context.Context, userID string, data []byte) (media.BrokerPhoto, error) {
	f.gotUser, f.gotData = userID, data
	return media.BrokerPhoto{PhotoPath: "/public/media/brokers/x.jpg"}, f.err
}
func (f *fakeService) RemoveBrokerPhoto(_ context.Context, userID string) error {
	f.removedBy = userID
	return f.err
}
func (f *fakeService) BrokerPhoto(_ context.Context, file string) (storage.Object, error) {
	f.gotFile = file
	return f.object(), f.err
}
func (f *fakeService) UploadLeadPhoto(_ context.Context, token string, data []byte) (media.Photo, error) {
	f.gotToken, f.gotData = token, data
	return media.Photo{ID: "p1"}, f.err
}
func (f *fakeService) LeadPhoto(_ context.Context, userID, leadID, photoID string) (storage.Object, error) {
	f.gotUser, f.gotLead, f.gotPhoto = userID, leadID, photoID
	return f.object(), f.err
}
func (f *fakeService) object() storage.Object {
	if f.err != nil {
		return storage.Object{}
	}
	return storage.Object{Body: io.NopCloser(strings.NewReader("img")), ContentType: "image/jpeg", Size: 3}
}

type fakeLog struct{}

func (fakeLog) Error(string, ...any) {}

func router(svc *fakeService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewMediaHandler(svc, fakeLog{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.PUT("/me/photo", h.SetBrokerPhoto)
	r.DELETE("/me/photo", h.RemoveBrokerPhoto)
	r.GET("/public/media/brokers/:file", h.BrokerPhoto)
	r.POST("/public/lead/photos", h.UploadLeadPhoto)
	r.GET("/me/leads/:id/photos/:photoID", h.MyLeadPhoto)
	r.GET("/admin/leads/:id/photos/:photoID", h.AdminLeadPhoto)
	return r
}

func multipartBody(t *testing.T, field string, data []byte) (*bytes.Buffer, string) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile(field, "foto.jpg")
	assert.NoError(t, err)
	_, _ = fw.Write(data)
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

func do(r *gin.Engine, method, path string, body io.Reader, contentType string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUploads(t *testing.T) {
	svc := &fakeService{}
	body, ct := multipartBody(t, "file", []byte("abc"))
	w := do(router(svc), http.MethodPut, "/me/photo", body, ct, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "/public/media/brokers/x.jpg")
	assert.Equal(t, "u1", svc.gotUser)
	assert.Equal(t, []byte("abc"), svc.gotData)

	body, ct = multipartBody(t, "file", []byte("xyz"))
	w = do(router(svc), http.MethodPost, "/public/lead/photos", body, ct, map[string]string{TokenHeader: "tok"})
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "tok", svc.gotToken)

	body, ct = multipartBody(t, "outro", []byte("abc"))
	assert.Equal(t, http.StatusBadRequest, do(router(svc), http.MethodPut, "/me/photo", body, ct, nil).Code, "sem o campo file")

	body, ct = multipartBody(t, "file", make([]byte, media.MaxBrokerPhotoBytes+1))
	assert.Equal(t, http.StatusRequestEntityTooLarge, do(router(svc), http.MethodPut, "/me/photo", body, ct, nil).Code)

	body, ct = multipartBody(t, "file", make([]byte, media.MaxBrokerPhotoBytes+200<<10))
	assert.Equal(t, http.StatusRequestEntityTooLarge, do(router(svc), http.MethodPut, "/me/photo", body, ct, nil).Code, "corpo acima do limite")
}

func TestRemoveAndServe(t *testing.T) {
	svc := &fakeService{}
	assert.Equal(t, http.StatusOK, do(router(svc), http.MethodDelete, "/me/photo", nil, "", nil).Code)
	assert.Equal(t, "u1", svc.removedBy)

	w := do(router(svc), http.MethodGet, "/public/media/brokers/a.jpg", nil, "", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "img", w.Body.String())
	assert.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Cache-Control"), "immutable")
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "a.jpg", svc.gotFile)

	w = do(router(svc), http.MethodGet, "/me/leads/l1/photos/p1", nil, "", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Cache-Control"), "private")
	assert.Equal(t, []string{"u1", "l1", "p1"}, []string{svc.gotUser, svc.gotLead, svc.gotPhoto})

	assert.Equal(t, http.StatusOK, do(router(svc), http.MethodGet, "/admin/leads/l2/photos/p2", nil, "", nil).Code)
	assert.Equal(t, []string{"", "l2", "p2"}, []string{svc.gotUser, svc.gotLead, svc.gotPhoto}, "o admin vê sem filtro de corretor")
}

func TestErrors(t *testing.T) {
	for err, status := range map[error]int{
		media.ErrNotFound:       http.StatusNotFound,
		media.ErrInvalidToken:   http.StatusUnauthorized,
		media.ErrBrokerNotFound: http.StatusForbidden,
		media.ErrTooLarge:       http.StatusRequestEntityTooLarge,
		media.ErrNotImage:       http.StatusBadRequest,
		media.ErrTooMany:        http.StatusBadRequest,
		media.ErrUnavailable:    http.StatusServiceUnavailable,
		errors.New("db"):        http.StatusInternalServerError,
	} {
		svc := &fakeService{err: err}
		body, ct := multipartBody(t, "file", []byte("abc"))
		assert.Equal(t, status, do(router(svc), http.MethodPut, "/me/photo", body, ct, nil).Code, err.Error())
		assert.Equal(t, status, do(router(svc), http.MethodGet, "/public/media/brokers/a.jpg", nil, "", nil).Code, err.Error())
	}
}
