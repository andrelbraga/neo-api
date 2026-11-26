package route

import (
	"neo360-api/internal/domain/media"
	"neo360-api/internal/domain/media/handler"
	"neo360-api/internal/domain/media/repository"
	"neo360-api/internal/domain/media/service"
	"neo360-api/internal/infra/api"
	"neo360-api/internal/pkg/authz"
)

// Register expõe as fotos (ADR-0029). bucket nil (sem S3 configurado) responde que o envio não está disponível.
func Register(app *api.App, bucket media.Bucket) {
	h := handler.NewMediaHandler(service.NewMediaService(repository.NewMediaRepository(app.DB()), bucket, app.Logger()), app.Logger())

	public := app.API(api.Routes{})
	public.GET("/public/media/brokers/:file", h.BrokerPhoto)
	public.POST("/public/lead/photos", h.UploadLeadPhoto)

	me := app.API(api.Routes{Authorization: true, Permission: authz.BrokerArea})
	me.PUT("/me/photo", h.SetBrokerPhoto)
	me.DELETE("/me/photo", h.RemoveBrokerPhoto)
	me.GET("/me/leads/:id/photos/:photoID", h.MyLeadPhoto)

	admin := app.API(api.Routes{Authorization: true, Permission: authz.Admin})
	admin.GET("/admin/leads/:id/photos/:photoID", h.AdminLeadPhoto)
}
