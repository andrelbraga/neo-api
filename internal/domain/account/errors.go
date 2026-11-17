package account

import "errors"

var (
	// ErrInvalidCredentials não diz se o e-mail existe: a mesma resposta para e-mail ou senha errados.
	ErrInvalidCredentials = errors.New("e-mail ou senha inválidos")
	ErrNotFound           = errors.New("conta não encontrada")
	ErrEmailTaken         = errors.New("já existe uma conta com este e-mail")
	ErrWeakPassword       = errors.New("a senha precisa de pelo menos 8 caracteres")
	ErrSamePassword       = errors.New("a nova senha precisa ser diferente da atual")
	ErrWrongPassword      = errors.New("a senha atual não confere")
)
