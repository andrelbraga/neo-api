package broker

import "errors"

var (
	ErrNotFound       = errors.New("corretor não encontrado")
	ErrNameRequired   = errors.New("informe nome e sobrenome")
	ErrNameTooLong    = errors.New("nome ou sobrenome longo demais (até 80 caracteres)")
	ErrInvalidPhone   = errors.New("telefone inválido: informe DDD e número")
	ErrInvalidCreci   = errors.New("CRECI inválido (até 20 caracteres)")
	ErrInvalidEmail   = errors.New("e-mail inválido")
	ErrWeakPassword   = errors.New("a senha precisa de pelo menos 8 caracteres")
	ErrEmailTaken     = errors.New("já existe uma conta com este e-mail")
	ErrAgencyNotFound = errors.New("imobiliária não encontrada")
	// ErrCodeTaken é interno: o serviço sorteia outro código.
	ErrCodeTaken = errors.New("código de corretor em uso")
)
