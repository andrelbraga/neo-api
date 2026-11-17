package account

import "errors"

var (
	// ErrInvalidCredentials não diz se o e-mail existe: a mesma resposta para e-mail ou senha errados.
	ErrInvalidCredentials = errors.New("e-mail ou senha inválidos")
	ErrNotFound           = errors.New("conta não encontrada")
	ErrEmailTaken         = errors.New("já existe uma conta com este e-mail")
)
