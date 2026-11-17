package route

import (
	"neo360-api/internal/domain/account"
	"neo360-api/internal/domain/account/handler"
	"neo360-api/internal/domain/account/repository"
	"neo360-api/internal/domain/account/service"
	"neo360-api/internal/infra/api"
	"neo360-api/internal/pkg/authz"
	"neo360-api/internal/pkg/security"
)

// Register expõe entrar, o cadastro aberto do corretor, quem sou eu, o perfil do corretor, a troca de senha e,
// no admin, a lista de usuários com o reset por senha provisória (ADR-0020).
func Register(app *api.App, brokers account.Brokers) {
	repo := repository.NewAccountRepository(app.DB())
	svc := service.NewAccountService(repo, brokers, app.Tokens(), security.NewBcryptHasher())
	h := handler.NewAccountHandler(svc, app.Logger())

	public := app.API(api.Routes{})
	public.POST("/auth/login", h.Login)
	public.POST("/auth/signup", h.Signup)

	auth := app.API(api.Routes{Authorization: true})
	auth.GET("/auth/me", h.Me)
	auth.PUT("/auth/password", h.ChangePassword)
	admin := app.API(api.Routes{Authorization: true, Permission: authz.Admin})
	admin.GET("/admin/users", h.Accounts)
	admin.POST("/admin/users/:id/reset-password", h.ResetPassword)
	app.API(api.Routes{Authorization: true, Permission: authz.BrokerArea}).PUT("/me/profile", h.UpdateProfile)
}
