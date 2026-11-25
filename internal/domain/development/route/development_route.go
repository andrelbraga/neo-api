package route

import (
	"neo360-api/internal/domain/development"
	"neo360-api/internal/domain/development/handler"
	"neo360-api/internal/infra/api"
	"neo360-api/internal/pkg/authz"
)

// Register expõe o cadastro de empreendimentos no admin e a lista dos publicados para o corretor.
func Register(app *api.App, svc development.Service) {
	h := handler.NewDevelopmentHandler(svc, app.Logger())

	admin := app.API(api.Routes{Authorization: true, Permission: authz.Admin})
	admin.GET("/admin/developments", h.List)
	admin.POST("/admin/developments", h.Create)
	admin.GET("/admin/developments/:id", h.Get)
	admin.PUT("/admin/developments/:id", h.Update)
	admin.PUT("/admin/developments/:id/page", h.SavePage)

	app.API(api.Routes{Authorization: true, Permission: authz.BrokerArea}).GET("/me/developments", h.ListPublished)
}
