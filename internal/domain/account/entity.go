// Package account é a conta de acesso: entrar, o cadastro aberto do corretor (RN-01.1), quem sou eu
// e o perfil que o corretor mantém.
package account

import (
	"strings"
	"time"
	"unicode/utf8"

	"neo360-api/internal/domain/broker"
)

type User struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Email       string         `json:"email"`
	Role        string         `json:"role"`
	Permissions []string       `json:"permissions"`
	Broker      *broker.Broker `json:"broker,omitempty"`
	// MustChangePassword: o admin redefiniu a senha e o usuário precisa criar a dele (ADR-0020).
	MustChangePassword bool `json:"must_change_password"`
}

// Account é a conta na lista de usuários do admin. A senha nunca sai: só existe o hash, que não se desfaz.
type Account struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Email              string     `json:"email"`
	Role               string     `json:"role"`
	Active             bool       `json:"active"`
	BrokerCode         string     `json:"broker_code"`
	MustChangePassword bool       `json:"must_change_password"`
	PasswordResetAt    *time.Time `json:"password_reset_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

// TemporaryPassword é a senha provisória gerada no reset: aparece uma vez para o admin e não fica guardada.
type TemporaryPassword struct {
	Password string `json:"temporary_password"`
}

type PasswordChange struct {
	Current string `json:"current_password"`
	New     string `json:"new_password"`
}

// Validate confere a senha nova: o mesmo mínimo do cadastro e diferente da atual.
func (r PasswordChange) Validate() error {
	switch {
	case utf8.RuneCountInString(r.New) < minPassword:
		return ErrWeakPassword
	case r.New == r.Current:
		return ErrSamePassword
	}
	return nil
}

// Credentials é o que o login confere; nunca sai da API.
// minPassword é o mesmo mínimo do cadastro do corretor (broker.CreateRequest).
const minPassword = 8

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
