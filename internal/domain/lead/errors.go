package lead

import "errors"

var (
	ErrNotFound         = errors.New("cliente não encontrado")
	ErrShowroomNotFound = errors.New("link não encontrado ou desativado")
	ErrInvalidToken     = errors.New("visita não iniciada: abra o link do corretor de novo")
	ErrInvalidEvent     = errors.New("evento inválido")
	ErrInvalidName      = errors.New("informe seu nome")
	ErrInvalidPhone     = errors.New("WhatsApp inválido: informe DDD e número")
	ErrInvalidProposal  = errors.New("proposta inválida: confira os valores e os textos (resumo até 900 e observação até 400 caracteres)")
	ErrInvalidSale      = errors.New("venda fechada precisa do valor; a observação vai até 400 caracteres")
	ErrInvalidStage     = errors.New("etapa inválida: novo, em atendimento, proposta, venda fechada ou sem venda")
	ErrNotBroker        = errors.New("esta conta não é de corretor")
)
