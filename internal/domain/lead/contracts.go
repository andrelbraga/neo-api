package lead

import (
	"context"

	"neo360-api/internal/domain/development"
)

type Repository interface {
	ShowroomBroker(ctx context.Context, code string) (PublicBroker, error)
	BrokerIDByUser(ctx context.Context, userID string) (string, error)
	FindByToken(ctx context.Context, tokenHash string) (Lead, error)
	Create(ctx context.Context, developmentID, brokerID, tokenHash, layout string) (Lead, error)
	AddEvent(ctx context.Context, leadID, kind string, data []byte) error
	SetContact(ctx context.Context, leadID string, r ContactRequest, eventData []byte) error
	Get(ctx context.Context, id string) (Lead, error)
	List(ctx context.Context, f Filter) ([]Lead, error)
	Events(ctx context.Context, leadID string) ([]Event, error)
	SetStage(ctx context.Context, id string, r StageRequest) (Lead, error)
	Proposals(ctx context.Context, leadID string) ([]Proposal, error)
	// AddProposal grava a proposta e, se é do cliente, o evento "proposta"; leva o cliente à etapa proposta
	// se ele ainda estava em novo ou em atendimento.
	AddProposal(ctx context.Context, leadID string, p Proposal) (Proposal, error)
	MarkProposalsSeen(ctx context.Context, leadID string) error
	// MergeByPhone junta os cadastros do mesmo WhatsApp no mesmo empreendimento no mais antigo e devolve
	// o id que ficou (ADR-0016). Sem duplicado, devolve o próprio leadID.
	MergeByPhone(ctx context.Context, leadID string) (string, error)
	Summary(ctx context.Context, brokerID string) (Summary, error)
	Activity(ctx context.Context, brokerID string) ([]Activity, error)

	// Cliente cadastrado pelo corretor (RN-03.16).
	// DevelopmentPublished devolve ErrInvalidDevelopment se o empreendimento não existe ou não está publicado.
	DevelopmentPublished(ctx context.Context, developmentID string) error
	// FindByPhone acha o cliente (não juntado) com este WhatsApp no empreendimento.
	FindByPhone(ctx context.Context, developmentID, phone string) (Lead, error)
	CreateByBroker(ctx context.Context, brokerID, tokenHash, inviteCode string, c NewClientRequest) (Lead, error)
	// FindByInvite acha o cliente cadastrado pelo código do link de convite, no empreendimento.
	FindByInvite(ctx context.Context, developmentID, code string) (Lead, error)
	// AddAlias liga um novo token de navegador ao cliente (um cadastro juntado a ele) e, se o cliente ainda
	// não tinha visita, grava o layout desta primeira visita.
	AddAlias(ctx context.Context, leadID, tokenHash, layout string) error

	// Edição, arquivamento e modelos de mensagem (ADR-0025).
	// Update troca nome, WhatsApp e perfil; eventData não vazio grava o evento de edição com os dados de antes.
	Update(ctx context.Context, id string, r UpdateRequest, eventData []byte) (Lead, error)
	SetArchived(ctx context.Context, id string, archived bool) (Lead, error)
	// Templates devolve os textos que o corretor mudou, por perfil e tom.
	Templates(ctx context.Context, brokerID string) ([]Template, error)
	SaveTemplate(ctx context.Context, brokerID string, t Template) error
	DeleteTemplate(ctx context.Context, brokerID, profile, tone string) error
	// MessageSent grava o evento de mensagem enviada e leva o cliente de novo para em atendimento (RN-03.22).
	MessageSent(ctx context.Context, id string, eventData []byte) (Lead, error)
}

// Developments é o que o showroom usa do empreendimento: só o publicado, pelo endereço.
type Developments interface {
	GetPublishedBySlug(ctx context.Context, slug string) (development.Development, error)
}

// PublicService atende o cliente no showroom, identificado pelo token da visita.
type PublicService interface {
	Showroom(ctx context.Context, slug, code string) (Showroom, error)
	// invite é o código do link de convite de um cliente cadastrado pelo corretor (RN-03.16); pode vir vazio.
	Visit(ctx context.Context, slug, code, token, layout, invite string) (Visit, error)
	Track(ctx context.Context, token string, r EventRequest) error
	Contact(ctx context.Context, token string, r ContactRequest) (Visit, error)
	Propose(ctx context.Context, token string, r ProposalRequest) (Proposal, error)
}

// BrokerService é o CRM do corretor: só os clientes de que ele é dono.
type BrokerService interface {
	List(ctx context.Context, userID string, f Filter) ([]Lead, error)
	Get(ctx context.Context, userID, id string) (Detail, error)
	SetStage(ctx context.Context, userID, id string, r StageRequest) (Lead, error)
	Adjust(ctx context.Context, userID, id string, r AdjustRequest) (Proposal, error)
	Summary(ctx context.Context, userID string) (Summary, error)
	Activity(ctx context.Context, userID string) ([]Activity, error)
	// Create cadastra um cliente à mão; Import, os da planilha CSV (RN-03.16). Com dryRun, Import só confere.
	Create(ctx context.Context, userID string, r NewClientRequest) (Lead, error)
	Import(ctx context.Context, userID, developmentID string, data []byte, dryRun bool) (ImportReport, error)
	// Update edita nome, WhatsApp e perfil; Archive e Restore são o "excluir" sem apagar nada (ADR-0025).
	Update(ctx context.Context, userID, id string, r UpdateRequest) (Lead, error)
	Archive(ctx context.Context, userID, id string) (Lead, error)
	Restore(ctx context.Context, userID, id string) (Lead, error)
	// Templates são os modelos de mensagem do corretor: os dele e, no resto, o padrão.
	Templates(ctx context.Context, userID string) ([]Template, error)
	SaveTemplate(ctx context.Context, userID, profile, tone string, r TemplateRequest) (Template, error)
	// ResetTemplate volta o modelo ao padrão.
	ResetTemplate(ctx context.Context, userID, profile, tone string) (Template, error)
	// MessageSent registra o envio confirmado pelo corretor na fila de envio (ADR-0028).
	MessageSent(ctx context.Context, userID, id string, r MessageSentRequest) (Lead, error)
}

// AdminService é a visão de tudo no admin, sem marcar propostas como vistas.
type AdminService interface {
	List(ctx context.Context, f Filter) ([]Lead, error)
	Get(ctx context.Context, id string) (Detail, error)
	// brokerID vazio é a plataforma inteira; preenchido, é o "ver como" do admin (RN-01.10).
	Summary(ctx context.Context, brokerID string) (Summary, error)
	Activity(ctx context.Context, brokerID string) ([]Activity, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
