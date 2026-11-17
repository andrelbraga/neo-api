package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"neo360-api/internal/pkg/security"
)

// ErrAccountNotFound é o usuário do token que não existe mais ou foi desativado.
var ErrAccountNotFound = errors.New("account not found or inactive")

type TokenParser interface {
	Parse(token string) (security.Claims, error)
}

// Accounts confere o usuário do token no banco: devolve o papel atual de quem está ativo.
type Accounts interface {
	ActiveRole(ctx context.Context, userID string) (string, error)
}

// PasswordGate diz se o usuário está com senha provisória (ADR-0020). Opcional: quem implementa Accounts
// e PasswordGate faz o middleware barrar o resto da API até a troca.
type PasswordGate interface {
	MustChangePassword(ctx context.Context, userID string) (bool, error)
}

// Com senha provisória, só estas rotas respondem: ver a própria conta e trocar a senha.
var allowedWithTemporaryPassword = map[string]bool{"/api/auth/me": true, "/api/auth/password": true}

// AuthMiddleware exige "Authorization: Bearer <jwt>" e injeta userID e role no contexto (RN-01.5).
// Com accounts, o papel vem do banco a cada requisição: quem foi desativado perde o acesso na hora
// e a troca de papel vale sem novo login (RN-01.5).
func AuthMiddleware(tokens TokenParser, accounts Accounts) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || scheme != "Bearer" || strings.TrimSpace(token) == "" {
			abort(c, http.StatusUnauthorized, "token de acesso ausente ou mal formatado")
			return
		}

		claims, err := tokens.Parse(strings.TrimSpace(token))
		if err != nil {
			abort(c, http.StatusUnauthorized, "token inválido ou expirado")
			return
		}

		role := claims.Role
		if accounts != nil {
			role, err = accounts.ActiveRole(c.Request.Context(), claims.UserID)
			if errors.Is(err, ErrAccountNotFound) {
				abort(c, http.StatusUnauthorized, "sessão encerrada; entre de novo")
				return
			}
			if err != nil {
				abort(c, http.StatusInternalServerError, "erro interno")
				return
			}
		}

		if gate, ok := accounts.(PasswordGate); ok && !allowedWithTemporaryPassword[c.FullPath()] {
			must, err := gate.MustChangePassword(c.Request.Context(), claims.UserID)
			if err != nil {
				abort(c, http.StatusInternalServerError, "erro interno")
				return
			}
			if must {
				abort(c, http.StatusForbidden, "troque a senha provisória para continuar")
				return
			}
		}

		c.Set("userID", claims.UserID)
		c.Set("role", role)
		c.Next()
	}
}

func abort(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"status": status, "data": gin.H{"error": message}})
}
