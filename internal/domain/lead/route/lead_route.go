package route

import (
	"neo360-api/internal/domain/lead"
	"neo360-api/internal/domain/lead/handler"
	"neo360-api/internal/domain/lead/repository"
	"neo360-api/internal/domain/lead/service"
	"neo360-api/internal/infra/api"
	"neo360-api/internal/pkg/authz"
)

// Register expõe o showroom público (com o envio de proposta), o CRM do corretor e a visão do admin.
func Register(app *api.App, developments lead.Developments) {
	repo := repository.NewLeadRepository(app.DB())
	h := handler.NewLeadHandler(service.NewPublicService(repo, developments), service.NewBrokerService(repo),
		service.NewAdminService(repo), app.Logger())

	public := app.API(api.Routes{})
	public.GET("/public/showroom/:slug/:code", h.Showroom)
	public.POST("/public/showroom/:slug/:code/visit", h.Visit)
	public.POST("/public/lead/events", h.Track)
	public.PUT("/public/lead/contact", h.Contact)
	public.POST("/public/lead/proposals", h.Propose)

	me := app.API(api.Routes{Authorization: true, Permission: authz.BrokerArea})
	me.GET("/me/leads", h.MyLeads)
	me.GET("/me/leads/:id", h.MyLead)
	me.PATCH("/me/leads/:id/stage", h.SetStage)
	me.POST("/me/leads/:id/proposals", h.Adjust)
	me.GET("/me/summary", h.MySummary)
	me.GET("/me/activity", h.MyActivity)

	admin := app.API(api.Routes{Authorization: true, Permission: authz.Admin})
	admin.GET("/admin/leads", h.AdminLeads)
	admin.GET("/admin/leads/:id", h.AdminLead)
	admin.GET("/admin/summary", h.AdminSummary)
	admin.GET("/admin/activity", h.AdminActivity)
}
