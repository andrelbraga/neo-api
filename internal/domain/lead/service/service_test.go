package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/development"
	"neo360-api/internal/domain/lead"
)

// memRepo é um repositório em memória com dois corretores: u-fab (b-fab, FAB-0001) e u-ana (b-ana, ANA-0002).
type memRepo struct {
	leads     map[string]*lead.Lead
	byHash    map[string]string
	events    map[string][]string
	proposals map[string][]lead.Proposal
	seen      []string
	stages    []lead.StageRequest
	failOn    string
}

func newMem() *memRepo {
	return &memRepo{leads: map[string]*lead.Lead{}, byHash: map[string]string{}, events: map[string][]string{}, proposals: map[string][]lead.Proposal{}}
}

func (m *memRepo) fail(op string) error {
	if m.failOn == op {
		return errors.New(op)
	}
	return nil
}

func (m *memRepo) ShowroomBroker(_ context.Context, code string) (lead.PublicBroker, error) {
	switch code {
	case "FAB-0001":
		return lead.PublicBroker{ID: "b-fab", Code: code, Name: "Fabiano"}, m.fail("broker")
	case "ANA-0002":
		return lead.PublicBroker{ID: "b-ana", Code: code, Name: "Ana"}, m.fail("broker")
	}
	return lead.PublicBroker{}, lead.ErrShowroomNotFound
}
func (m *memRepo) BrokerIDByUser(_ context.Context, userID string) (string, error) {
	switch userID {
	case "u-fab":
		return "b-fab", nil
	case "u-ana":
		return "b-ana", nil
	}
	return "", lead.ErrNotBroker
}
func (m *memRepo) FindByToken(_ context.Context, h string) (lead.Lead, error) {
	if err := m.fail("find"); err != nil {
		return lead.Lead{}, err
	}
	id, ok := m.byHash[h]
	if !ok {
		return lead.Lead{}, lead.ErrNotFound
	}
	return *m.leads[id], nil
}
func (m *memRepo) Create(_ context.Context, devID, brokerID, h, layout string) (lead.Lead, error) {
	if err := m.fail("create"); err != nil {
		return lead.Lead{}, err
	}
	id := fmt.Sprintf("l%d", len(m.leads)+1)
	m.leads[id] = &lead.Lead{ID: id, DevelopmentID: devID, BrokerID: brokerID, Stage: lead.StageNew, Layout: layout}
	m.byHash[h] = id
	return *m.leads[id], nil
}
func (m *memRepo) AddEvent(_ context.Context, id, kind string, data []byte) error {
	m.events[id] = append(m.events[id], kind+string(data))
	return m.fail("event")
}
func (m *memRepo) SetContact(_ context.Context, id string, r lead.ContactRequest, data []byte) error {
	m.leads[id].Name, m.leads[id].Phone = r.Name, r.Phone
	m.events[id] = append(m.events[id], "contato"+string(data))
	return m.fail("contact")
}
func (m *memRepo) Get(_ context.Context, id string) (lead.Lead, error) {
	l, ok := m.leads[id]
	if !ok {
		return lead.Lead{}, lead.ErrNotFound
	}
	return *l, nil
}
func (m *memRepo) List(_ context.Context, f lead.Filter) ([]lead.Lead, error) {
	out := []lead.Lead{}
	for _, l := range m.leads {
		if (f.BrokerID == "" || l.BrokerID == f.BrokerID) && (f.Stage == "" || l.Stage == f.Stage) {
			out = append(out, *l)
		}
	}
	return out, nil
}
func (m *memRepo) Events(_ context.Context, id string) ([]lead.Event, error) {
	out := []lead.Event{}
	for i, e := range m.events[id] {
		out = append(out, lead.Event{ID: int64(i), Kind: e})
	}
	return out, m.fail("events")
}
func (m *memRepo) SetStage(_ context.Context, id string, r lead.StageRequest) (lead.Lead, error) {
	m.leads[id].Stage = r.Stage
	m.stages = append(m.stages, r)
	return *m.leads[id], nil
}
func (m *memRepo) Proposals(_ context.Context, id string) ([]lead.Proposal, error) {
	return m.proposals[id], m.fail("proposals")
}
func (m *memRepo) AddProposal(_ context.Context, id string, p lead.Proposal) (lead.Proposal, error) {
	if err := m.fail("add-proposal"); err != nil {
		return lead.Proposal{}, err
	}
	p.ID = fmt.Sprintf("p%d", len(m.proposals[id])+1)
	m.proposals[id] = append(m.proposals[id], p)
	if st := m.leads[id].Stage; st == lead.StageNew || st == lead.StageWorking {
		m.leads[id].Stage = lead.StageProposal
	}
	return p, nil
}

// MergeByPhone junta no cadastro mais antigo (menor id) os outros do mesmo telefone e empreendimento.
func (m *memRepo) MergeByPhone(_ context.Context, id string) (string, error) {
	if err := m.fail("merge"); err != nil {
		return "", err
	}
	l := m.leads[id]
	kept := id
	for oid, o := range m.leads {
		if o.Phone == l.Phone && o.DevelopmentID == l.DevelopmentID && oid < kept {
			kept = oid
		}
	}
	for oid, o := range m.leads {
		if oid != kept && o.Phone == l.Phone && o.DevelopmentID == l.DevelopmentID {
			m.proposals[kept] = append(m.proposals[kept], m.proposals[oid]...)
			delete(m.proposals, oid)
			for h, target := range m.byHash {
				if target == oid {
					m.byHash[h] = kept
				}
			}
			delete(m.leads, oid)
		}
	}
	return kept, nil
}
func (m *memRepo) MarkProposalsSeen(_ context.Context, id string) error {
	m.seen = append(m.seen, id)
	return m.fail("seen")
}
func (m *memRepo) Activity(_ context.Context, brokerID string) ([]lead.Activity, error) {
	out := []lead.Activity{}
	for _, l := range m.leads {
		if brokerID == "" || l.BrokerID == brokerID {
			out = append(out, lead.Activity{LeadID: l.ID})
		}
	}
	return out, m.fail("activity")
}
func (m *memRepo) Summary(_ context.Context, brokerID string) (lead.Summary, error) {
	n := 0
	for _, l := range m.leads {
		if brokerID == "" || l.BrokerID == brokerID {
			n++
		}
	}
	return lead.Summary{Leads: n}, nil
}

type fakeDevs struct{ err error }

func (f fakeDevs) GetPublishedBySlug(_ context.Context, slug string) (development.Development, error) {
	if slug != "boa-vista" {
		return development.Development{}, development.ErrNotFound
	}
	return development.Development{ID: "d-" + slug, Slug: slug}, f.err
}

func newPublic(repo *memRepo) *PublicService {
	s := NewPublicService(repo, fakeDevs{})
	n := 0
	s.newToken = func() (string, error) { n++; return fmt.Sprintf("tok%d", n), nil }
	return s
}

func TestShowroom(t *testing.T) {
	ctx := context.Background()
	s := newPublic(newMem())
	room, err := s.Showroom(ctx, "boa-vista", " fab-0001 ")
	require.NoError(t, err)
	assert.Equal(t, "d-boa-vista", room.Development.ID)
	assert.Equal(t, "Fabiano", room.Broker.Name, "o código não diferencia maiúsculas")

	_, err = s.Showroom(ctx, "nao-publicado", "FAB-0001")
	assert.ErrorIs(t, err, lead.ErrShowroomNotFound)
	_, err = s.Showroom(ctx, "boa-vista", "XXX-9999")
	assert.ErrorIs(t, err, lead.ErrShowroomNotFound)

	s = NewPublicService(newMem(), fakeDevs{err: errors.New("db")})
	_, err = s.Showroom(ctx, "boa-vista", "FAB-0001")
	assert.EqualError(t, err, "db")
}

// RN-03.1: o dono é o corretor do link da primeira visita e não muda.
func TestVisit_OwnerIsFirstTouch(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newPublic(repo)

	v, err := s.Visit(ctx, "boa-vista", "FAB-0001", "", "")
	require.NoError(t, err)
	assert.Equal(t, "tok1", v.Token)
	assert.Equal(t, "b-fab", repo.leads["l1"].BrokerID)
	assert.Equal(t, []string{`visita{"codigo":"FAB-0001"}`}, repo.events["l1"])
	assert.NotContains(t, repo.byHash, "tok1", "o banco guarda só o hash do token")

	v, err = s.Visit(ctx, "boa-vista", "ANA-0002", "tok1", "")
	require.NoError(t, err)
	assert.Equal(t, "tok1", v.Token, "mesmo cliente")
	assert.Len(t, repo.leads, 1)
	assert.Equal(t, "b-fab", repo.leads["l1"].BrokerID, "chegou pelo link da Ana, mas continua do Fabiano")
	assert.Equal(t, `visita{"codigo":"ANA-0002"}`, repo.events["l1"][1], "a visita guarda por qual link ele voltou")

	v, err = s.Visit(ctx, "boa-vista", "ANA-0002", "token-desconhecido", "")
	require.NoError(t, err)
	assert.Equal(t, "tok2", v.Token, "token que não existe: cliente novo")
	assert.Equal(t, "b-ana", repo.leads["l2"].BrokerID)
}

// ADR-0019: o cliente novo guarda o layout em que chegou; quem volta mantém o da primeira visita.
func TestVisit_Layout(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newPublic(repo)
	_, err := s.Visit(ctx, "boa-vista", "FAB-0001", "", " B ")
	require.NoError(t, err)
	assert.Equal(t, "b", repo.leads["l1"].Layout)
	_, err = s.Visit(ctx, "boa-vista", "FAB-0001", "tok1", "a")
	require.NoError(t, err)
	assert.Equal(t, "b", repo.leads["l1"].Layout, "volta pelo outro layout, mas conta no da primeira visita")
	_, err = s.Visit(ctx, "boa-vista", "FAB-0001", "", "qualquer")
	require.NoError(t, err)
	assert.Equal(t, "a", repo.leads["l2"].Layout, "valor desconhecido é o layout A")
}

func TestVisit_ReturnsContactAndOtherDevelopment(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newPublic(repo)
	_, err := s.Visit(ctx, "boa-vista", "FAB-0001", "", "")
	require.NoError(t, err)
	repo.leads["l1"].Name, repo.leads["l1"].Phone = "Maria", "54999990000"
	v, err := s.Visit(ctx, "boa-vista", "FAB-0001", "tok1", "")
	require.NoError(t, err)
	assert.Equal(t, lead.Visit{Token: "tok1", Name: "Maria", Phone: "54999990000"}, v, "o nome volta para a página")

	repo.leads["l1"].DevelopmentID = "outro"
	v, err = s.Visit(ctx, "boa-vista", "FAB-0001", "tok1", "")
	require.NoError(t, err)
	assert.Equal(t, "tok2", v.Token, "token de outro empreendimento vira cliente novo")
}

func TestVisit_Errors(t *testing.T) {
	ctx := context.Background()
	_, err := newPublic(newMem()).Visit(ctx, "x", "FAB-0001", "", "")
	assert.ErrorIs(t, err, lead.ErrShowroomNotFound)

	for _, op := range []string{"find", "create", "event"} {
		repo := newMem()
		s := newPublic(repo)
		token := ""
		if op == "event" {
			_, err := s.Visit(ctx, "boa-vista", "FAB-0001", "", "")
			require.NoError(t, err)
			token = "tok1"
		}
		if op == "find" {
			token = "qualquer"
		}
		repo.failOn = op
		_, err := s.Visit(ctx, "boa-vista", "FAB-0001", token, "")
		assert.EqualError(t, err, op)
	}

	repo := newMem()
	repo.failOn = "event"
	_, err = newPublic(repo).Visit(ctx, "boa-vista", "FAB-0001", "", "")
	assert.EqualError(t, err, "event", "evento da primeira visita")

	s := NewPublicService(newMem(), fakeDevs{})
	s.newToken = func() (string, error) { return "", errors.New("rand") }
	_, err = s.Visit(ctx, "boa-vista", "FAB-0001", "", "")
	assert.EqualError(t, err, "rand")
}

func TestTrackAndContact(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newPublic(repo)
	_, err := s.Visit(ctx, "boa-vista", "FAB-0001", "", "")
	require.NoError(t, err)

	require.NoError(t, s.Track(ctx, "tok1", lead.EventRequest{Kind: "tour_3d", Data: json.RawMessage(`{"torre":"A"}`)}))
	assert.Equal(t, `tour_3d{"torre":"A"}`, repo.events["l1"][1])
	assert.ErrorIs(t, s.Track(ctx, "tok1", lead.EventRequest{Kind: "x"}), lead.ErrInvalidEvent)
	assert.ErrorIs(t, s.Track(ctx, " ", lead.EventRequest{Kind: "secao"}), lead.ErrInvalidToken)
	assert.ErrorIs(t, s.Track(ctx, "outro", lead.EventRequest{Kind: "secao"}), lead.ErrInvalidToken)

	v, err := s.Contact(ctx, "tok1", lead.ContactRequest{Name: "Maria", Phone: "(54) 99999-0000"})
	require.NoError(t, err)
	assert.Equal(t, "54999990000", v.Phone)
	assert.Equal(t, "Maria", repo.leads["l1"].Name)
	_, err = s.Contact(ctx, "tok1", lead.ContactRequest{})
	assert.ErrorIs(t, err, lead.ErrInvalidName)
	_, err = s.Contact(ctx, "x", lead.ContactRequest{Name: "M", Phone: "54999990000"})
	assert.ErrorIs(t, err, lead.ErrInvalidToken)
	assert.Equal(t, "contato{}", repo.events["l1"][2], "primeiro contato, sem dados anteriores")

	_, err = s.Contact(ctx, "tok1", lead.ContactRequest{Name: "Maria", Phone: "54999990000"})
	require.NoError(t, err)
	assert.Len(t, repo.events["l1"], 3, "reenviar os mesmos dados não gera evento")

	v, err = s.Contact(ctx, "tok1", lead.ContactRequest{Name: "Joana", Phone: "51911110000"})
	require.NoError(t, err)
	assert.Equal(t, "Joana", v.Name)
	assert.Equal(t, `contato{"antes":{"nome":"Maria","whatsapp":"54999990000"}}`, repo.events["l1"][3], "a troca guarda os dados anteriores")

	repo.failOn = "contact"
	_, err = s.Contact(ctx, "tok1", lead.ContactRequest{Name: "M", Phone: "54999990000"})
	assert.EqualError(t, err, "contact")
	repo.failOn = "merge"
	_, err = s.Contact(ctx, "tok1", lead.ContactRequest{Name: "Ana", Phone: "51922220000"})
	assert.EqualError(t, err, "merge")
}

// RN-03.14: o mesmo WhatsApp em outro navegador é o mesmo cliente; as propostas ficam juntas no mais antigo.
func TestPropose_SamePhoneSameClient(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newPublic(repo)
	_, err := s.Visit(ctx, "boa-vista", "FAB-0001", "", "")
	require.NoError(t, err)
	_, err = s.Visit(ctx, "boa-vista", "ANA-0002", "", "") // outro navegador, link de outra corretora
	require.NoError(t, err)
	req := func(summary string) lead.ProposalRequest {
		return lead.ProposalRequest{Name: "Maria", Phone: "54999990000", Summary: summary}
	}
	p1, err := s.Propose(ctx, "tok1", req("primeira"))
	require.NoError(t, err)
	assert.Equal(t, "54999990000", p1.ClientPhone)
	assert.Equal(t, "Maria", p1.ClientName)
	_, err = s.Propose(ctx, "tok2", req("segunda"))
	require.NoError(t, err)

	require.Len(t, repo.leads, 1, "um cliente só")
	assert.Equal(t, "b-fab", repo.leads["l1"].BrokerID, "fica com o dono do cadastro mais antigo")
	assert.Len(t, repo.proposals["l1"], 2, "as duas propostas no mesmo cliente")
	v, err := s.Visit(ctx, "boa-vista", "ANA-0002", "tok2", "")
	require.NoError(t, err)
	assert.Equal(t, "tok2", v.Token, "o token do outro navegador continua valendo")
}

// RN-03.8: a proposta chega com nome, WhatsApp e os números, e leva o cliente à etapa proposta.
func TestPropose(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newPublic(repo)
	_, err := s.Visit(ctx, "boa-vista", "FAB-0001", "", "")
	require.NoError(t, err)

	req := lead.ProposalRequest{Name: "Maria", Phone: "(54) 99999-0000", Summary: "entrada de R$ 150 mil", Note: " carro na entrada ", WantsContract: true,
		Condition: json.RawMessage(`{"preco":370000,"entrada":150000}`)}
	p, err := s.Propose(ctx, "tok1", req)
	require.NoError(t, err)
	assert.Equal(t, lead.AuthorClient, p.Author)
	assert.Equal(t, "carro na entrada", p.Note)
	assert.True(t, p.WantsContract)
	assert.Equal(t, "Maria", repo.leads["l1"].Name, "a proposta grava o contato")
	assert.Equal(t, lead.StageProposal, repo.leads["l1"].Stage)

	_, err = s.Propose(ctx, "tok1", lead.ProposalRequest{Name: "Maria", Phone: "54999990000"})
	assert.ErrorIs(t, err, lead.ErrInvalidProposal, "sem resumo e sem números")
	_, err = s.Propose(ctx, "tok1", lead.ProposalRequest{Phone: "54999990000", Summary: "x"})
	assert.ErrorIs(t, err, lead.ErrInvalidName)
	_, err = s.Propose(ctx, "", req)
	assert.ErrorIs(t, err, lead.ErrInvalidToken)
	repo.failOn = "add-proposal"
	_, err = s.Propose(ctx, "tok1", req)
	assert.EqualError(t, err, "add-proposal")
}

func TestBrokerService(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	pub := newPublic(repo)
	_, _ = pub.Visit(ctx, "boa-vista", "FAB-0001", "", "")
	_, _ = pub.Visit(ctx, "boa-vista", "ANA-0002", "", "")
	s := NewBrokerService(repo)

	list, err := s.List(ctx, "u-fab", lead.Filter{BrokerID: "b-ana"})
	require.NoError(t, err)
	require.Len(t, list, 1, "o filtro de corretor vem da conta, não do pedido")
	assert.Equal(t, "l1", list[0].ID)
	_, err = s.List(ctx, "u-fab", lead.Filter{Stage: "x"})
	assert.ErrorIs(t, err, lead.ErrInvalidStage)
	_, err = s.List(ctx, "admin", lead.Filter{})
	assert.ErrorIs(t, err, lead.ErrNotBroker)

	repo.leads["l1"].NewProposals = 2
	d, err := s.Get(ctx, "u-fab", "l1")
	require.NoError(t, err)
	assert.Len(t, d.Events, 1)
	assert.Zero(t, d.NewProposals)
	assert.Equal(t, []string{"l1"}, repo.seen, "abrir a ficha marca as propostas como vistas")
	_, err = s.Get(ctx, "u-fab", "l2")
	assert.ErrorIs(t, err, lead.ErrNotFound, "cliente de outro corretor")
	_, err = s.Get(ctx, "u-fab", "l9")
	assert.ErrorIs(t, err, lead.ErrNotFound)
	_, err = s.Get(ctx, "admin", "l1")
	assert.ErrorIs(t, err, lead.ErrNotBroker)

	l, err := s.SetStage(ctx, "u-fab", "l1", lead.StageRequest{Stage: lead.StageWorking, SaleValue: 9, Note: "x"})
	require.NoError(t, err)
	assert.Equal(t, lead.StageWorking, l.Stage)
	assert.Equal(t, lead.StageRequest{Stage: lead.StageWorking}, repo.stages[0], "etapa em aberto não leva desfecho")
	_, err = s.SetStage(ctx, "u-fab", "l1", lead.StageRequest{Stage: "x"})
	assert.ErrorIs(t, err, lead.ErrInvalidStage)
	_, err = s.SetStage(ctx, "u-fab", "l2", lead.StageRequest{Stage: lead.StageWorking})
	assert.ErrorIs(t, err, lead.ErrNotFound)

	// RN-03.12: o corretor ajusta a proposta; o ajuste não apaga a anterior.
	repo.proposals["l1"] = []lead.Proposal{{ID: "p1", Author: lead.AuthorClient}}
	adj, err := s.Adjust(ctx, "u-fab", "l1", lead.AdjustRequest{BasedOn: "p1", Condition: json.RawMessage(`{"preco":360000}`), Note: "desconto à vista"})
	require.NoError(t, err)
	assert.Equal(t, lead.AuthorBroker, adj.Author)
	require.NotNil(t, adj.BasedOn)
	assert.Equal(t, "p1", *adj.BasedOn)
	assert.Len(t, repo.proposals["l1"], 2)
	_, err = s.Adjust(ctx, "u-fab", "l1", lead.AdjustRequest{Condition: json.RawMessage(`{"preco":0}`)})
	assert.ErrorIs(t, err, lead.ErrInvalidProposal, "ajuste sem preço")
	_, err = s.Adjust(ctx, "u-fab", "l1", lead.AdjustRequest{BasedOn: "p-de-outro", Condition: json.RawMessage(`{"preco":1}`)})
	assert.ErrorIs(t, err, lead.ErrInvalidProposal, "base de outro cliente")
	_, err = s.Adjust(ctx, "u-fab", "l2", lead.AdjustRequest{Condition: json.RawMessage(`{"preco":1}`)})
	assert.ErrorIs(t, err, lead.ErrNotFound)

	// RN-03.13: venda fechada exige valor e guarda a proposta de base.
	_, err = s.SetStage(ctx, "u-fab", "l1", lead.StageRequest{Stage: lead.StageSold})
	assert.ErrorIs(t, err, lead.ErrInvalidSale)
	_, err = s.SetStage(ctx, "u-fab", "l1", lead.StageRequest{Stage: lead.StageSold, SaleValue: 360000, ProposalID: "nao-existe"})
	assert.ErrorIs(t, err, lead.ErrInvalidProposal)
	l, err = s.SetStage(ctx, "u-fab", "l1", lead.StageRequest{Stage: lead.StageSold, SaleValue: 360000, ProposalID: "p2", Note: " assinado "})
	require.NoError(t, err)
	assert.Equal(t, lead.StageSold, l.Stage)
	assert.Equal(t, lead.StageRequest{Stage: lead.StageSold, SaleValue: 360000, ProposalID: "p2", Note: "assinado"}, repo.stages[len(repo.stages)-1])
	_, err = s.SetStage(ctx, "u-fab", "l1", lead.StageRequest{Stage: lead.StageLost, SaleValue: 1, ProposalID: "p2", Note: "comprou outro"})
	require.NoError(t, err)
	assert.Equal(t, lead.StageRequest{Stage: lead.StageLost, Note: "comprou outro"}, repo.stages[len(repo.stages)-1], "sem venda guarda só o motivo")

	sum, err := s.Summary(ctx, "u-fab")
	require.NoError(t, err)
	assert.Equal(t, 1, sum.Leads)
	_, err = s.Summary(ctx, "admin")
	assert.ErrorIs(t, err, lead.ErrNotBroker)

	feed, err := s.Activity(ctx, "u-fab")
	require.NoError(t, err)
	assert.Len(t, feed, 1, "só os clientes do corretor")
	_, err = s.Activity(ctx, "admin")
	assert.ErrorIs(t, err, lead.ErrNotBroker)

	for _, op := range []string{"seen", "events", "proposals"} {
		repo.failOn = op
		_, err = s.Get(ctx, "u-fab", "l1")
		assert.EqualError(t, err, op)
	}
	repo.failOn = "proposals"
	_, err = s.Adjust(ctx, "u-fab", "l1", lead.AdjustRequest{BasedOn: "p1", Condition: json.RawMessage(`{"preco":1}`)})
	assert.EqualError(t, err, "proposals")
}

func TestAdminService(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	pub := newPublic(repo)
	_, _ = pub.Visit(ctx, "boa-vista", "FAB-0001", "", "")
	_, _ = pub.Visit(ctx, "boa-vista", "ANA-0002", "", "")
	s := NewAdminService(repo)

	list, err := s.List(ctx, lead.Filter{})
	require.NoError(t, err)
	assert.Len(t, list, 2, "o admin vê todos")
	_, err = s.List(ctx, lead.Filter{Stage: "x"})
	assert.ErrorIs(t, err, lead.ErrInvalidStage)

	d, err := s.Get(ctx, "l2")
	require.NoError(t, err)
	assert.Equal(t, "b-ana", d.BrokerID)
	assert.Empty(t, repo.seen, "o admin não marca as propostas como vistas")
	_, err = s.Get(ctx, "l9")
	assert.ErrorIs(t, err, lead.ErrNotFound)

	sum, err := s.Summary(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, sum.Leads)

	feed, err := s.Activity(ctx)
	require.NoError(t, err)
	assert.Len(t, feed, 2)
}
