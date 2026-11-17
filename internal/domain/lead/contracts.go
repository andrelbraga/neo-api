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
}

// Developments é o que o showroom usa do empreendimento: só o publicado, pelo endereço.
type Developments interface {
	GetPublishedBySlug(ctx context.Context, slug string) (development.Development, error)
}

// PublicService atende o cliente no showroom, identificado pelo token da visita.
type PublicService interface {
	Showroom(ctx context.Context, slug, code string) (Showroom, error)
	Visit(ctx context.Context, slug, code, token, layout string) (Visit, error)
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
}

// AdminService é a visão de tudo no admin, sem marcar propostas como vistas.
type AdminService interface {
	List(ctx context.Context, f Filter) ([]Lead, error)
	Get(ctx context.Context, id string) (Detail, error)
	Summary(ctx context.Context) (Summary, error)
	Activity(ctx context.Context) ([]Activity, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
