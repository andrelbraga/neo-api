// Package account é a conta de acesso: entrar, o cadastro aberto do corretor (RN-01.1), quem sou eu
// e o perfil que o corretor mantém.
package account

import (
	"strings"
	"time"

	"neo360-api/internal/domain/broker"
)

type User struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Email       string         `json:"email"`
	Role        string         `json:"role"`
	Permissions []string       `json:"permissions"`
	Broker      *broker.Broker `json:"broker,omitempty"`
}

// Credentials é o que o login confere; nunca sai da API.
type Credentials struct {
	UserID       string
	Role         string
	PasswordHash string
	Active       bool
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r LoginRequest) Normalize() LoginRequest {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	return r
}

type Session struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}
