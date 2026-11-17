package route

import (
	"neo360-api/internal/domain/broker"
	"neo360-api/internal/domain/broker/handler"
	"neo360-api/internal/infra/api"
	"neo360-api/internal/pkg/authz"
)

// Register expõe o cadastro de profissionais no admin. O mesmo serviço atende o cadastro aberto da conta.
func Register(app *api.App, svc broker.Service) {
	h := handler.NewBrokerHandler(svc, app.Logger())
	g := app.API(api.Routes{Authorization: true, Permission: authz.Admin})
	g.GET("/admin/brokers", h.List)
	g.POST("/admin/brokers", h.Create)
	g.GET("/admin/brokers/:id", h.Get)
	g.PUT("/admin/brokers/:id", h.Update)
}
