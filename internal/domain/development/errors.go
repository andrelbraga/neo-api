package development

import "errors"

var (
	ErrNotFound          = errors.New("empreendimento não encontrado")
	ErrNameRequired      = errors.New("informe o nome do empreendimento")
	ErrInvalidSlug       = errors.New("endereço inválido: use letras minúsculas, números e hífen")
	ErrSlugTaken         = errors.New("já existe um empreendimento com este endereço")
	ErrInvalidStatus     = errors.New("situação inválida: lançamento, em obras ou pronto")
	ErrInvalidState      = errors.New("UF inválida: use a sigla com duas letras")
	ErrInvalidURL        = errors.New("endereço de imagem ou do tour inválido: use https:// ou um caminho do próprio app")
	ErrInvalidHighlight  = errors.New("cada destaque precisa de nome e valor")
	ErrTooManyItems      = errors.New("muitos itens na galeria ou nos destaques")
	ErrFieldTooLong      = errors.New("campo longo demais")
	ErrDeveloperNotFound = errors.New("incorporadora não encontrada")
)
