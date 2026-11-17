package organization

import "errors"

var (
	ErrNotFound     = errors.New("empresa não encontrada")
	ErrInvalidKind  = errors.New("tipo inválido: incorporadora ou imobiliária")
	ErrNameRequired = errors.New("informe o nome da empresa")
	ErrFieldTooLong = errors.New("campo longo demais")
	ErrInvalidEmail = errors.New("e-mail inválido")
	ErrInvalidLogo  = errors.New("logo inválido: use https:// ou um caminho do próprio app")
)
