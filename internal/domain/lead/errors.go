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

	// Cliente cadastrado pelo corretor (RN-03.16).
	ErrInvalidClientName  = errors.New("informe o nome do cliente (até 80 caracteres)")
	ErrInvalidDevelopment = errors.New("escolha um empreendimento publicado")
	ErrAlreadyYours       = errors.New("este WhatsApp já é seu cliente neste empreendimento")
	ErrPhoneTaken         = errors.New("este WhatsApp já está cadastrado neste empreendimento")
	ErrInvalidFile        = errors.New("arquivo inválido: envie uma planilha CSV com as colunas nome e telefone")
	ErrTooManyRows        = errors.New("a planilha passa do limite de 500 clientes; divida em arquivos menores")

	// Edição, arquivamento e modelos de mensagem (ADR-0025).
	ErrInvalidProfile  = errors.New("perfil inválido: moradia, investimento, veraneio ou sem perfil")
	ErrInvalidTemplate = errors.New("escreva a mensagem (até 1.000 caracteres)")
	ErrTemplateKey     = errors.New("modelo não encontrado")
)
