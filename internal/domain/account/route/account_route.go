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

// Register expõe entrar, o cadastro aberto do corretor, quem sou eu e o perfil do corretor.
func Register(app *api.App, brokers account.Brokers) {
	repo := repository.NewAccountRepository(app.DB())
	svc := service.NewAccountService(repo, brokers, app.Tokens(), security.NewBcryptHasher())
	h := handler.NewAccountHandler(svc, app.Logger())

	public := app.API(api.Routes{})
	public.POST("/auth/login", h.Login)
	public.POST("/auth/signup", h.Signup)

	app.API(api.Routes{Authorization: true}).GET("/auth/me", h.Me)
	app.API(api.Routes{Authorization: true, Permission: authz.BrokerArea}).PUT("/me/profile", h.UpdateProfile)
}
