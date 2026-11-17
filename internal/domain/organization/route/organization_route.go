package route

import (
	"neo360-api/internal/domain/organization/handler"
	"neo360-api/internal/domain/organization/repository"
	"neo360-api/internal/domain/organization/service"
	"neo360-api/internal/infra/api"
	"neo360-api/internal/pkg/authz"
)

// Register expõe o cadastro de incorporadoras e imobiliárias no admin.
func Register(app *api.App) {
	repo := repository.NewOrganizationRepository(app.DB())
	h := handler.NewOrganizationHandler(service.NewOrganizationService(repo), app.Logger())
	g := app.API(api.Routes{Authorization: true, Permission: authz.Admin})
	g.GET("/admin/organizations", h.List)
	g.POST("/admin/organizations", h.Create)
	g.GET("/admin/organizations/:id", h.Get)
	g.PUT("/admin/organizations/:id", h.Update)
}
