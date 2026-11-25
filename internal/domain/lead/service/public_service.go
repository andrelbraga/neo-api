package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"neo360-api/internal/domain/development"
	"neo360-api/internal/domain/lead"
)

type PublicService struct {
	repo         lead.Repository
	developments lead.Developments
	newToken     func() (string, error)
}

func NewPublicService(repo lead.Repository, developments lead.Developments) *PublicService {
	return &PublicService{repo: repo, developments: developments, newToken: randomToken}
}

// Showroom devolve o empreendimento publicado e o corretor ativo do código (RN-02.2).
func (s *PublicService) Showroom(ctx context.Context, slug, code string) (lead.Showroom, error) {
	d, err := s.developments.GetPublishedBySlug(ctx, slug)
	if errors.Is(err, development.ErrNotFound) {
		return lead.Showroom{}, lead.ErrShowroomNotFound
	}
	if err != nil {
		return lead.Showroom{}, err
	}
	b, err := s.repo.ShowroomBroker(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return lead.Showroom{}, err
	}
	return lead.Showroom{Development: d, Broker: b}, nil
}

// Visit registra a abertura do link. Com um token já conhecido para o mesmo empreendimento, o cliente
// continua com o dono de antes, mesmo que tenha chegado pelo link de outro corretor (RN-03.1). O cliente
// novo guarda o layout da apresentação em que chegou; quem volta mantém o da primeira visita (ADR-0019).
//
// Com o código de um convite (RN-03.16), a visita cai na ficha do cliente que o corretor cadastrou: o
// navegador ganha um token ligado a ela. Convite que não existe vale como o link do corretor.
func (s *PublicService) Visit(ctx context.Context, slug, code, token, layout, invite string) (lead.Visit, error) {
	room, err := s.Showroom(ctx, slug, code)
	if err != nil {
		return lead.Visit{}, err
	}
	visitData, _ := json.Marshal(map[string]string{"codigo": room.Broker.Code})
	token = strings.TrimSpace(token)
	if invite = strings.TrimSpace(invite); invite != "" {
		v, ok, err := s.visitInvite(ctx, room.Development.ID, invite, token, layout, visitData)
		if ok || err != nil {
			return v, err
		}
	}
	if token != "" {
		l, err := s.repo.FindByToken(ctx, hash(token))
		switch {
		case err == nil && l.DevelopmentID == room.Development.ID:
			if err := s.repo.AddEvent(ctx, l.ID, lead.EventVisit, visitData); err != nil {
				return lead.Visit{}, err
			}
			return lead.Visit{Token: token, Name: l.Name, Phone: l.Phone}, nil
		case err != nil && !errors.Is(err, lead.ErrNotFound):
			return lead.Visit{}, err
		}
	}
	token, err = s.newToken()
	if err != nil {
		return lead.Visit{}, err
	}
	l, err := s.repo.Create(ctx, room.Development.ID, room.Broker.ID, hash(token), lead.NormalizeLayout(layout))
	if err != nil {
		return lead.Visit{}, err
	}
	if err := s.repo.AddEvent(ctx, l.ID, lead.EventVisit, visitData); err != nil {
		return lead.Visit{}, err
	}
	return lead.Visit{Token: token}, nil
}

// visitInvite abre a ficha do cliente convidado. ok é falso quando o convite não existe neste empreendimento.
func (s *PublicService) visitInvite(ctx context.Context, developmentID, invite, token, layout string, visitData []byte) (lead.Visit, bool, error) {
	l, err := s.repo.FindByInvite(ctx, developmentID, invite)
	if errors.Is(err, lead.ErrNotFound) {
		return lead.Visit{}, false, nil
	}
	if err != nil {
		return lead.Visit{}, false, err
	}
	// O navegador que já abriu este convite continua com o mesmo token.
	known := token != ""
	if known {
		current, err := s.repo.FindByToken(ctx, hash(token))
		if err != nil && !errors.Is(err, lead.ErrNotFound) {
			return lead.Visit{}, false, err
		}
		known = err == nil && current.ID == l.ID
	}
	if !known {
		if token, err = s.newToken(); err != nil {
			return lead.Visit{}, false, err
		}
		if err := s.repo.AddAlias(ctx, l.ID, hash(token), lead.NormalizeLayout(layout)); err != nil {
			return lead.Visit{}, false, err
		}
	}
	if err := s.repo.AddEvent(ctx, l.ID, lead.EventVisit, visitData); err != nil {
		return lead.Visit{}, false, err
	}
	return lead.Visit{Token: token, Name: l.Name, Phone: l.Phone}, true, nil
}

func (s *PublicService) Track(ctx context.Context, token string, r lead.EventRequest) error {
	valid, err := r.Validate()
	if err != nil {
		return err
	}
	l, err := s.byToken(ctx, token)
	if err != nil {
		return err
	}
	return s.repo.AddEvent(ctx, l.ID, valid.Kind, valid.Data)
}

// Contact grava nome e WhatsApp que o cliente informou para o corretor retornar (RN-03.4). O cliente
// pode trocar os dados a qualquer hora; a troca guarda os anteriores no evento, para o corretor não perder
// o contato que já usava, e o reenvio dos mesmos dados não gera evento (RN-03.10).
func (s *PublicService) Contact(ctx context.Context, token string, r lead.ContactRequest) (lead.Visit, error) {
	valid, err := r.Validate()
	if err != nil {
		return lead.Visit{}, err
	}
	l, err := s.byToken(ctx, token)
	if err != nil {
		return lead.Visit{}, err
	}
	visit := lead.Visit{Token: token, Name: valid.Name, Phone: valid.Phone}
	if l.Name == valid.Name && l.Phone == valid.Phone {
		return visit, nil
	}
	data := []byte(`{}`)
	if l.Name != "" || l.Phone != "" {
		data, _ = json.Marshal(map[string]map[string]string{"antes": {"nome": l.Name, "whatsapp": l.Phone}})
	}
	if err := s.repo.SetContact(ctx, l.ID, valid, data); err != nil {
		return lead.Visit{}, err
	}
	// Mesmo WhatsApp em outro navegador: é o mesmo cliente, com as propostas juntas (ADR-0016).
	if _, err := s.repo.MergeByPhone(ctx, l.ID); err != nil {
		return lead.Visit{}, err
	}
	return visit, nil
}

// Propose grava a proposta que o cliente montou na apresentação, com o nome e o WhatsApp para o corretor
// retornar (RN-03.8). Propostas do mesmo WhatsApp ficam no mesmo cliente, mesmo vindas de outro navegador
// (RN-03.14). A conversa segue pelo WhatsApp; a proposta fica no CRM, com os números.
func (s *PublicService) Propose(ctx context.Context, token string, r lead.ProposalRequest) (lead.Proposal, error) {
	valid, err := r.Validate()
	if err != nil {
		return lead.Proposal{}, err
	}
	if _, err := s.Contact(ctx, token, lead.ContactRequest{Name: valid.Name, Phone: valid.Phone}); err != nil {
		return lead.Proposal{}, err
	}
	l, err := s.byToken(ctx, token)
	if err != nil {
		return lead.Proposal{}, err
	}
	return s.repo.AddProposal(ctx, l.ID, lead.Proposal{
		Author: lead.AuthorClient, Condition: valid.Condition, Summary: valid.Summary, Note: valid.Note, WantsContract: valid.WantsContract,
		ClientName: valid.Name, ClientPhone: valid.Phone,
	})
}

func (s *PublicService) byToken(ctx context.Context, token string) (lead.Lead, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return lead.Lead{}, lead.ErrInvalidToken
	}
	l, err := s.repo.FindByToken(ctx, hash(token))
	if errors.Is(err, lead.ErrNotFound) {
		return lead.Lead{}, lead.ErrInvalidToken
	}
	return l, err
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
