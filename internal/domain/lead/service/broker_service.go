package service

import (
	"context"

	"neo360-api/internal/domain/lead"
)

// BrokerService é o CRM do corretor. Todo acesso passa pelo dono: cliente de outro corretor é "não encontrado".
type BrokerService struct {
	repo lead.Repository
	// newToken e newCode geram o token guardado (sem dono) e o código do convite do cliente cadastrado.
	newToken func() (string, error)
	newCode  func() (string, error)
}

func NewBrokerService(repo lead.Repository) *BrokerService {
	return &BrokerService{repo: repo, newToken: randomToken, newCode: inviteCode}
}

func (s *BrokerService) List(ctx context.Context, userID string, f lead.Filter) ([]lead.Lead, error) {
	brokerID, err := s.repo.BrokerIDByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if f.Stage != "" && !lead.ValidStage(f.Stage) {
		return nil, lead.ErrInvalidStage
	}
	f.BrokerID = brokerID
	return s.repo.List(ctx, f)
}

// Get abre a ficha do cliente e marca como vistas as propostas dele.
func (s *BrokerService) Get(ctx context.Context, userID, id string) (lead.Detail, error) {
	l, err := s.own(ctx, userID, id)
	if err != nil {
		return lead.Detail{}, err
	}
	if err := s.repo.MarkProposalsSeen(ctx, l.ID); err != nil {
		return lead.Detail{}, err
	}
	l.NewProposals = 0
	return detail(ctx, s.repo, l)
}

// SetStage muda a etapa. Venda fechada guarda valor, proposta de base e observação; sem venda guarda o
// motivo; voltar a uma etapa em aberto reabre a negociação (RN-03.13). Só o dono muda.
func (s *BrokerService) SetStage(ctx context.Context, userID, id string, r lead.StageRequest) (lead.Lead, error) {
	valid, err := r.Validate()
	if err != nil {
		return lead.Lead{}, err
	}
	l, err := s.own(ctx, userID, id)
	if err != nil {
		return lead.Lead{}, err
	}
	if err := s.ownProposal(ctx, l.ID, valid.ProposalID); err != nil {
		return lead.Lead{}, err
	}
	return s.repo.SetStage(ctx, l.ID, valid)
}

// Adjust grava um ajuste do corretor sobre a proposta (RN-03.12). A nova versão não apaga a anterior.
func (s *BrokerService) Adjust(ctx context.Context, userID, id string, r lead.AdjustRequest) (lead.Proposal, error) {
	valid, err := r.Validate()
	if err != nil {
		return lead.Proposal{}, err
	}
	l, err := s.own(ctx, userID, id)
	if err != nil {
		return lead.Proposal{}, err
	}
	if err := s.ownProposal(ctx, l.ID, valid.BasedOn); err != nil {
		return lead.Proposal{}, err
	}
	p := lead.Proposal{Author: lead.AuthorBroker, Condition: valid.Condition, Summary: valid.Summary, Note: valid.Note}
	if valid.BasedOn != "" {
		p.BasedOn = &valid.BasedOn
	}
	return s.repo.AddProposal(ctx, l.ID, p)
}

func (s *BrokerService) Summary(ctx context.Context, userID string) (lead.Summary, error) {
	brokerID, err := s.repo.BrokerIDByUser(ctx, userID)
	if err != nil {
		return lead.Summary{}, err
	}
	return s.repo.Summary(ctx, brokerID)
}

// Activity é o feed dos clientes do corretor: o que fizeram por último.
func (s *BrokerService) Activity(ctx context.Context, userID string) ([]lead.Activity, error) {
	brokerID, err := s.repo.BrokerIDByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.Activity(ctx, brokerID)
}

func (s *BrokerService) own(ctx context.Context, userID, id string) (lead.Lead, error) {
	brokerID, err := s.repo.BrokerIDByUser(ctx, userID)
	if err != nil {
		return lead.Lead{}, err
	}
	l, err := s.repo.Get(ctx, id)
	if err != nil {
		return lead.Lead{}, err
	}
	if l.BrokerID != brokerID {
		return lead.Lead{}, lead.ErrNotFound
	}
	return l, nil
}

// ownProposal confere que a proposta citada (base de ajuste ou de venda) é deste cliente. Vazio passa.
func (s *BrokerService) ownProposal(ctx context.Context, leadID, proposalID string) error {
	if proposalID == "" {
		return nil
	}
	list, err := s.repo.Proposals(ctx, leadID)
	if err != nil {
		return err
	}
	for _, p := range list {
		if p.ID == proposalID {
			return nil
		}
	}
	return lead.ErrInvalidProposal
}

func detail(ctx context.Context, repo lead.Repository, l lead.Lead) (lead.Detail, error) {
	events, err := repo.Events(ctx, l.ID)
	if err != nil {
		return lead.Detail{}, err
	}
	proposals, err := repo.Proposals(ctx, l.ID)
	if err != nil {
		return lead.Detail{}, err
	}
	return lead.Detail{Lead: l, Events: events, Proposals: proposals}, nil
}
