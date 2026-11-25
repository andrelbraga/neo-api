//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/lead"
	"neo360-api/internal/testutil"
)

func TestLeadRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewLeadRepository(database)
	ctx := context.Background()

	var devID, userID, brokerID, otherUser, otherBroker string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT id FROM developments WHERE slug = 'boa-vista-residencial'`).Scan(&devID))
	newBroker := func(email, code string, user, broker *string) {
		require.NoError(t, database.QueryRowContext(ctx,
			`INSERT INTO users (name, email, password_hash, role) VALUES ('X', $1, 'h', 'corretor') RETURNING id`, email).Scan(user))
		require.NoError(t, database.QueryRowContext(ctx,
			`INSERT INTO brokers (user_id, first_name, last_name, phone, creci, code) VALUES ($1, 'Fabiano', 'Souza', '54999990000', '123', $2) RETURNING id`,
			*user, code).Scan(broker))
	}
	newBroker("fab@x.com", "FAB-0001", &userID, &brokerID)
	newBroker("ana@x.com", "ANA-0001", &otherUser, &otherBroker)

	b, err := repo.ShowroomBroker(ctx, "FAB-0001")
	require.NoError(t, err)
	assert.Equal(t, lead.PublicBroker{ID: brokerID, Name: "Fabiano Souza", Phone: "54999990000", Creci: "123", Code: "FAB-0001"}, b)
	var agencyID string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT id FROM organizations WHERE kind = 'imobiliaria' AND name = 'Omni'`).Scan(&agencyID))
	_, err = database.ExecContext(ctx, `UPDATE brokers SET agency_id = $1 WHERE id = $2`, agencyID, otherBroker)
	require.NoError(t, err)
	ana, err := repo.ShowroomBroker(ctx, "ANA-0001")
	require.NoError(t, err)
	assert.Equal(t, "Omni", ana.AgencyName, "o showroom mostra a imobiliária do corretor")
	assert.Equal(t, "/logos/imobiliaria-omni.png", ana.AgencyLogo, "com o logo da imobiliária pré-cadastrada")
	_, err = repo.ShowroomBroker(ctx, "XXX-0000")
	assert.ErrorIs(t, err, lead.ErrShowroomNotFound)

	id, err := repo.BrokerIDByUser(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, brokerID, id)
	for _, u := range []string{"00000000-0000-0000-0000-000000000000", "nao-e-uuid"} {
		_, err = repo.BrokerIDByUser(ctx, u)
		assert.ErrorIs(t, err, lead.ErrNotBroker)
	}

	l, err := repo.Create(ctx, devID, brokerID, "hash-1", "b")
	require.NoError(t, err)
	assert.Equal(t, lead.StageNew, l.Stage)
	assert.Equal(t, "b", l.Layout, "o layout da primeira visita fica no cliente")
	assert.Equal(t, "Boa Vista Residencial", l.DevelopmentName)
	assert.Equal(t, "FAB-0001", l.BrokerCode)
	other, err := repo.Create(ctx, devID, otherBroker, "hash-2", "a")
	require.NoError(t, err)

	found, err := repo.FindByToken(ctx, "hash-1")
	require.NoError(t, err)
	assert.Equal(t, l.ID, found.ID)
	_, err = repo.FindByToken(ctx, "nao-existe")
	assert.ErrorIs(t, err, lead.ErrNotFound)

	before := found.LastActivityAt
	require.NoError(t, repo.AddEvent(ctx, l.ID, lead.EventVisit, []byte(`{"codigo":"FAB-0001"}`)))
	require.NoError(t, repo.AddEvent(ctx, l.ID, "tour_3d", []byte(`{}`)))
	require.NoError(t, repo.SetContact(ctx, l.ID, lead.ContactRequest{Name: "Maria", Phone: "54977776666"}, []byte(`{}`)))
	got, err := repo.Get(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, "Maria", got.Name)
	assert.Equal(t, 3, got.EventsCount, "visita, tour e contato")
	assert.ElementsMatch(t, []string{"visita", "tour_3d", "contato"}, got.Signals)
	assert.Equal(t, []string{}, other.Signals, "sem eventos, lista vazia (não nula)")
	assert.False(t, got.LastActivityAt.Before(before))

	events, err := repo.Events(ctx, l.ID)
	require.NoError(t, err)
	require.Len(t, events, 3)
	assert.Equal(t, lead.EventVisit, events[0].Kind)
	assert.JSONEq(t, `{"codigo":"FAB-0001"}`, string(events[0].Data))
	assert.Equal(t, lead.EventContact, events[2].Kind)
	assert.JSONEq(t, `{}`, string(events[2].Data), "o primeiro contato não tem dados anteriores")

	require.NoError(t, repo.SetContact(ctx, l.ID, lead.ContactRequest{Name: "Maria Lima", Phone: "54977776666"},
		[]byte(`{"antes":{"nome":"Maria","whatsapp":"54977776666"}}`)))
	got, err = repo.Get(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, "Maria Lima", got.Name)
	events, err = repo.Events(ctx, l.ID)
	require.NoError(t, err)
	assert.JSONEq(t, `{"antes":{"nome":"Maria","whatsapp":"54977776666"}}`, string(events[3].Data), "a troca guarda os dados anteriores")

	// Proposta do cliente: registro, evento "proposta", etapa proposta e contagem de não vistas.
	p1, err := repo.AddProposal(ctx, l.ID, lead.Proposal{Author: lead.AuthorClient, Summary: "entrada de 150 mil", Note: "carro na entrada",
		WantsContract: true, Condition: []byte(`{"preco":370000,"entrada":150000}`)})
	require.NoError(t, err)
	assert.Equal(t, lead.AuthorClient, p1.Author)
	assert.Nil(t, p1.SeenAt)
	assert.JSONEq(t, `{"preco":370000,"entrada":150000}`, string(p1.Condition))
	got, err = repo.Get(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, lead.StageProposal, got.Stage, "a proposta leva o cliente à etapa proposta")
	assert.Equal(t, 1, got.NewProposals)
	assert.Contains(t, got.Signals, lead.EventProposal)
	events, err = repo.Events(ctx, l.ID)
	require.NoError(t, err)
	assert.JSONEq(t, `{"proposal_id":"`+p1.ID+`","resumo":"entrada de 150 mil","observacao":"carro na entrada","autorizou_contrato":true}`,
		string(events[len(events)-1].Data))

	sum, err := repo.Summary(ctx, brokerID)
	require.NoError(t, err)
	assert.Equal(t, lead.Summary{Leads: 1, Identified: 1, NewProposals: 1,
		ByStage:  map[string]int{lead.StageNew: 0, lead.StageWorking: 0, lead.StageProposal: 1, lead.StageSold: 0, lead.StageLost: 0},
		ByLayout: map[string]lead.LayoutStats{"a": {}, "b": {Leads: 1, WithProposal: 1}}}, sum, "o resumo traz a conversão por layout (ADR-0019)")

	require.NoError(t, repo.MarkProposalsSeen(ctx, l.ID))
	// Ajuste do corretor: não gera evento nem conta como nova.
	p2, err := repo.AddProposal(ctx, l.ID, lead.Proposal{Author: lead.AuthorBroker, BasedOn: &p1.ID, Note: "desconto", Condition: []byte(`{"preco":360000}`)})
	require.NoError(t, err)
	require.NotNil(t, p2.BasedOn)
	assert.Equal(t, p1.ID, *p2.BasedOn)
	proposals, err := repo.Proposals(ctx, l.ID)
	require.NoError(t, err)
	require.Len(t, proposals, 2)
	assert.Equal(t, p2.ID, proposals[0].ID, "mais recente primeiro")
	assert.NotNil(t, proposals[1].SeenAt, "a do cliente foi vista")
	got, err = repo.Get(ctx, l.ID)
	require.NoError(t, err)
	assert.Zero(t, got.NewProposals)
	_, err = repo.AddProposal(ctx, l.ID, lead.Proposal{Author: lead.AuthorBroker, BasedOn: ptr("nao-e-uuid"), Condition: []byte(`{}`)})
	assert.ErrorIs(t, err, lead.ErrNotFound, "base inválida")

	// Desfecho: venda fechada guarda valor e proposta; voltar para uma etapa em aberto limpa.
	upd, err := repo.SetStage(ctx, l.ID, lead.StageRequest{Stage: lead.StageSold, SaleValue: 360000, ProposalID: p2.ID, Note: "assinado"})
	require.NoError(t, err)
	assert.Equal(t, lead.StageSold, upd.Stage)
	require.NotNil(t, upd.SaleValue)
	assert.InDelta(t, 360000, *upd.SaleValue, 0.001)
	require.NotNil(t, upd.SaleProposalID)
	assert.Equal(t, p2.ID, *upd.SaleProposalID)
	assert.NotNil(t, upd.OutcomeAt)
	assert.Equal(t, "assinado", upd.OutcomeNote)
	sum, err = repo.Summary(ctx, brokerID)
	require.NoError(t, err)
	assert.Equal(t, 1, sum.Sales)
	assert.Equal(t, lead.LayoutStats{Leads: 1, WithProposal: 1, Sales: 1, SalesValue: 360000}, sum.ByLayout["b"], "conversão do layout B")
	assert.Equal(t, lead.LayoutStats{}, sum.ByLayout["a"])
	assert.InDelta(t, 360000, sum.SalesValue, 0.001)
	_, err = repo.AddProposal(ctx, l.ID, lead.Proposal{Author: lead.AuthorBroker, Condition: []byte(`{"preco":1}`)})
	require.NoError(t, err)
	got, err = repo.Get(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, lead.StageSold, got.Stage, "proposta nova não mexe no desfecho")

	upd, err = repo.SetStage(ctx, l.ID, lead.StageRequest{Stage: lead.StageLost, Note: "comprou outro"})
	require.NoError(t, err)
	assert.Nil(t, upd.SaleValue)
	assert.Equal(t, "comprou outro", upd.OutcomeNote)
	upd, err = repo.SetStage(ctx, l.ID, lead.StageRequest{Stage: lead.StageProposal})
	require.NoError(t, err)
	assert.Nil(t, upd.OutcomeAt, "reabrir limpa o desfecho")
	assert.Empty(t, upd.OutcomeNote)
	for _, bad := range []string{"00000000-0000-0000-0000-000000000000", "nao-e-uuid"} {
		_, err = repo.SetStage(ctx, bad, lead.StageRequest{Stage: lead.StageProposal})
		assert.ErrorIs(t, err, lead.ErrNotFound)
		_, err = repo.Get(ctx, bad)
		assert.ErrorIs(t, err, lead.ErrNotFound)
	}

	// ADR-0016: o mesmo WhatsApp em outro navegador (e por outro link) junta no cadastro mais antigo.
	l3, err := repo.Create(ctx, devID, otherBroker, "hash-3", "a")
	require.NoError(t, err)
	require.NoError(t, repo.SetContact(ctx, l3.ID, lead.ContactRequest{Name: "Maria Nova", Phone: "54977776666"}, []byte(`{}`)))
	p3, err := repo.AddProposal(ctx, l3.ID, lead.Proposal{Author: lead.AuthorClient, Summary: "outra", Condition: []byte(`{}`), ClientName: "Maria Nova", ClientPhone: "54977776666"})
	require.NoError(t, err)
	assert.Equal(t, "54977776666", p3.ClientPhone)
	kept, err := repo.MergeByPhone(ctx, l3.ID)
	require.NoError(t, err)
	assert.Equal(t, l.ID, kept, "fica o mais antigo")
	viaToken, err := repo.FindByToken(ctx, "hash-3")
	require.NoError(t, err)
	assert.Equal(t, l.ID, viaToken.ID, "o token do navegador juntado abre o cadastro que ficou")
	assert.Equal(t, "Maria Nova", viaToken.Name, "com o nome mais recente")
	assert.Equal(t, brokerID, viaToken.BrokerID, "e o dono da primeira visita")
	_, err = repo.Get(ctx, l3.ID)
	assert.ErrorIs(t, err, lead.ErrNotFound, "o juntado some das listas")
	merged, err := repo.Proposals(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, p3.ID, merged[0].ID, "a proposta veio junto")
	kept, err = repo.MergeByPhone(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, l.ID, kept, "sem duplicado, nada muda")
	_, err = repo.MergeByPhone(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, lead.ErrNotFound)

	mine, err := repo.List(ctx, lead.Filter{BrokerID: brokerID})
	require.NoError(t, err)
	require.Len(t, mine, 1)
	assert.Equal(t, l.ID, mine[0].ID)
	all, err := repo.List(ctx, lead.Filter{})
	require.NoError(t, err)
	assert.Len(t, all, 2)
	byStage, err := repo.List(ctx, lead.Filter{Stage: lead.StageNew})
	require.NoError(t, err)
	require.Len(t, byStage, 1)
	assert.Equal(t, other.ID, byStage[0].ID)
	byDev, err := repo.List(ctx, lead.Filter{DevelopmentID: devID})
	require.NoError(t, err)
	assert.Len(t, byDev, 2)
	none, err := repo.List(ctx, lead.Filter{BrokerID: "nao-e-uuid"})
	require.NoError(t, err)
	assert.Empty(t, none, "id malformado no filtro só não acha nada")

	require.NoError(t, repo.AddEvent(ctx, l.ID, "tempo", []byte(`{"h-neg":30}`)))
	require.NoError(t, repo.AddEvent(ctx, l.ID, "secao", []byte(`{"id":"h-neg"}`)))
	feed, err := repo.Activity(ctx, brokerID)
	require.NoError(t, err)
	require.Len(t, feed, 7, "seções vistas e tempo ficam fora do feed (com contato e proposta do cadastro juntado)")
	assert.Equal(t, lead.EventProposal, feed[0].Kind, "mais recente primeiro")
	assert.Equal(t, "Maria Nova", feed[0].LeadName, "nome mais recente, depois de juntar")
	allFeed, err := repo.Activity(ctx, "")
	require.NoError(t, err)
	assert.Len(t, allFeed, 7)

	total, err := repo.Summary(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, 2, total.Leads)

	_, err = database.ExecContext(ctx, `UPDATE users SET active = false WHERE id = $1`, userID)
	require.NoError(t, err)
	_, err = repo.ShowroomBroker(ctx, "FAB-0001")
	assert.ErrorIs(t, err, lead.ErrShowroomNotFound, "corretor desativado derruba o link")

	require.NoError(t, database.Close())
	_, err = repo.ShowroomBroker(ctx, "FAB-0001")
	assert.Error(t, err)
	_, err = repo.BrokerIDByUser(ctx, userID)
	assert.Error(t, err)
	_, err = repo.Create(ctx, devID, brokerID, "h3", "a")
	assert.Error(t, err)
	assert.Error(t, repo.AddEvent(ctx, l.ID, "x", []byte(`{}`)))
	assert.Error(t, repo.SetContact(ctx, l.ID, lead.ContactRequest{}, []byte(`{}`)))
	_, err = repo.List(ctx, lead.Filter{})
	assert.Error(t, err)
	_, err = repo.Events(ctx, l.ID)
	assert.Error(t, err)
	_, err = repo.SetStage(ctx, l.ID, lead.StageRequest{Stage: lead.StageNew})
	assert.Error(t, err)
	_, err = repo.Proposals(ctx, l.ID)
	assert.Error(t, err)
	_, err = repo.AddProposal(ctx, l.ID, lead.Proposal{Author: lead.AuthorClient, Condition: []byte(`{}`)})
	assert.Error(t, err)
	assert.Error(t, repo.MarkProposalsSeen(ctx, l.ID))
	_, err = repo.MergeByPhone(ctx, l.ID)
	assert.Error(t, err)
	_, err = repo.Summary(ctx, "")
	assert.Error(t, err)
	_, err = repo.Activity(ctx, "")
	assert.Error(t, err)
}

func ptr(s string) *string { return &s }

// RN-03.16: cliente cadastrado pelo corretor, o link de convite e a comparação de layouts.
func TestLeadRepository_ClientsByBroker(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewLeadRepository(database)
	ctx := context.Background()

	var devID, userID, brokerID, unpublished string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT id FROM developments WHERE slug = 'boa-vista-residencial'`).Scan(&devID))
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('X', 'fab@x.com', 'h', 'corretor') RETURNING id`).Scan(&userID))
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO brokers (user_id, first_name, last_name, phone, creci, code) VALUES ($1, 'Fabiano', 'Souza', '54999990000', '', 'FAB-0001') RETURNING id`,
		userID).Scan(&brokerID))
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO developments (slug, name, city, state, status, published) VALUES ('rascunho', 'Rascunho', 'Gramado', 'RS', 'lancamento', false) RETURNING id`).Scan(&unpublished))

	require.NoError(t, repo.DevelopmentPublished(ctx, devID))
	for _, id := range []string{unpublished, "00000000-0000-0000-0000-000000000000", "nao-e-uuid"} {
		assert.ErrorIs(t, repo.DevelopmentPublished(ctx, id), lead.ErrInvalidDevelopment, id)
	}

	c := lead.NewClientRequest{DevelopmentID: devID, Name: "Maria Souza", Phone: "54988887777"}
	l, err := repo.CreateByBroker(ctx, brokerID, "hash-guardado", "conv123", c)
	require.NoError(t, err)
	assert.Equal(t, lead.OriginBroker, l.Origin)
	assert.Equal(t, "conv123", l.InviteCode)
	assert.Equal(t, lead.StageNew, l.Stage)
	assert.Equal(t, "Maria Souza", l.Name)
	assert.Equal(t, "54988887777", l.Phone)
	_, err = repo.CreateByBroker(ctx, brokerID, "h2", "conv456", lead.NewClientRequest{DevelopmentID: "nao-e-uuid"})
	assert.ErrorIs(t, err, lead.ErrInvalidDevelopment)

	found, err := repo.FindByPhone(ctx, devID, "54988887777")
	require.NoError(t, err)
	assert.Equal(t, l.ID, found.ID)
	_, err = repo.FindByPhone(ctx, devID, "54900000000")
	assert.ErrorIs(t, err, lead.ErrNotFound)

	found, err = repo.FindByInvite(ctx, devID, "conv123")
	require.NoError(t, err)
	assert.Equal(t, l.ID, found.ID)
	_, err = repo.FindByInvite(ctx, unpublished, "conv123")
	assert.ErrorIs(t, err, lead.ErrNotFound, "o convite vale só no empreendimento dele")

	// Antes de abrir o link, o cliente cadastrado não entra na comparação de layouts.
	s, err := repo.Summary(ctx, brokerID)
	require.NoError(t, err)
	assert.Equal(t, 1, s.Leads)
	assert.Equal(t, 0, s.ByLayout["a"].Leads+s.ByLayout["b"].Leads)

	require.NoError(t, repo.AddAlias(ctx, l.ID, "hash-navegador", "b"))
	require.NoError(t, repo.AddEvent(ctx, l.ID, lead.EventVisit, []byte(`{}`)))
	byToken, err := repo.FindByToken(ctx, "hash-navegador")
	require.NoError(t, err)
	assert.Equal(t, l.ID, byToken.ID, "o token do navegador abre a ficha cadastrada")
	assert.Equal(t, "b", byToken.Layout, "a primeira visita grava o layout")
	require.NoError(t, repo.AddAlias(ctx, l.ID, "hash-outro-navegador", "a"))
	got, err := repo.Get(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, "b", got.Layout, "a segunda visita não troca o layout")

	s, err = repo.Summary(ctx, brokerID)
	require.NoError(t, err)
	assert.Equal(t, 1, s.Leads, "os cadastros do navegador não contam como clientes")
	assert.Equal(t, 1, s.ByLayout["b"].Leads, "depois de abrir, conta no layout")
	list, err := repo.List(ctx, lead.Filter{BrokerID: brokerID})
	require.NoError(t, err)
	assert.Len(t, list, 1)

	require.NoError(t, database.Close())
	assert.Error(t, repo.DevelopmentPublished(ctx, devID))
	_, err = repo.CreateByBroker(ctx, brokerID, "h3", "conv789", c)
	assert.Error(t, err)
	assert.Error(t, repo.AddAlias(ctx, l.ID, "h4", "a"))
}

// ADR-0025: edição, arquivamento e modelos de mensagem.
func TestLeadRepository_EditArchiveTemplates(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewLeadRepository(database)
	ctx := context.Background()

	var devID, userID, brokerID string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT id FROM developments WHERE slug = 'boa-vista-residencial'`).Scan(&devID))
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('X', 'fab@x.com', 'h', 'corretor') RETURNING id`).Scan(&userID))
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO brokers (user_id, first_name, last_name, phone, creci, code) VALUES ($1, 'Fabiano', 'Souza', '54999990000', '', 'FAB-0001') RETURNING id`,
		userID).Scan(&brokerID))
	l, err := repo.CreateByBroker(ctx, brokerID, "h1", "conv1", lead.NewClientRequest{DevelopmentID: devID, Name: "Maria", Phone: "54988887777"})
	require.NoError(t, err)
	assert.Equal(t, lead.ProfileNone, l.Profile)
	assert.Nil(t, l.ArchivedAt)

	got, err := repo.Update(ctx, l.ID, lead.UpdateRequest{Name: "Maria Souza", Phone: "54911112222", Profile: lead.ProfileVacation}, []byte(`{"antes":{"nome":"Maria"}}`))
	require.NoError(t, err)
	assert.Equal(t, "Maria Souza", got.Name)
	assert.Equal(t, "54911112222", got.Phone)
	assert.Equal(t, lead.ProfileVacation, got.Profile)
	events, err := repo.Events(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, lead.EventEdit, events[len(events)-1].Kind)
	_, err = repo.Update(ctx, l.ID, lead.UpdateRequest{Name: "Maria", Profile: "outro"}, nil)
	assert.Error(t, err, "o banco recusa perfil fora da lista")
	_, err = repo.Update(ctx, "00000000-0000-0000-0000-000000000000", lead.UpdateRequest{Name: "X"}, nil)
	assert.ErrorIs(t, err, lead.ErrNotFound)

	require.NoError(t, repo.AddEvent(ctx, l.ID, "tour_3d", []byte(`{}`)))
	archived, err := repo.SetArchived(ctx, l.ID, true)
	require.NoError(t, err)
	require.NotNil(t, archived.ArchivedAt)
	again, err := repo.SetArchived(ctx, l.ID, true)
	require.NoError(t, err)
	assert.Equal(t, archived.ArchivedAt.Unix(), again.ArchivedAt.Unix(), "arquivar de novo mantém a data")

	active, err := repo.List(ctx, lead.Filter{BrokerID: brokerID})
	require.NoError(t, err)
	assert.Empty(t, active, "arquivado sai da lista")
	old, err := repo.List(ctx, lead.Filter{BrokerID: brokerID, Archived: true})
	require.NoError(t, err)
	assert.Len(t, old, 1)
	s, err := repo.Summary(ctx, brokerID)
	require.NoError(t, err)
	assert.Equal(t, 0, s.Leads, "e dos números")
	feed, err := repo.Activity(ctx, brokerID)
	require.NoError(t, err)
	assert.Empty(t, feed, "e do feed")
	byPhone, err := repo.FindByPhone(ctx, devID, "54911112222")
	require.NoError(t, err)
	assert.Equal(t, l.ID, byPhone.ID, "o WhatsApp continua do corretor: nada se perde")

	_, err = repo.AddProposal(ctx, l.ID, lead.Proposal{Author: lead.AuthorClient, Condition: []byte(`{}`), Summary: "Quero"})
	require.NoError(t, err)
	back, err := repo.Get(ctx, l.ID)
	require.NoError(t, err)
	assert.Nil(t, back.ArchivedAt, "proposta nova do cliente tira do arquivo")

	_, err = repo.SetArchived(ctx, l.ID, true)
	require.NoError(t, err)
	restored, err := repo.SetArchived(ctx, l.ID, false)
	require.NoError(t, err)
	assert.Nil(t, restored.ArchivedAt)
	_, err = repo.SetArchived(ctx, "nao-e-uuid", true)
	assert.ErrorIs(t, err, lead.ErrNotFound)
	_, err = repo.SetArchived(ctx, "00000000-0000-0000-0000-000000000000", true)
	assert.ErrorIs(t, err, lead.ErrNotFound)

	list, err := repo.Templates(ctx, brokerID)
	require.NoError(t, err)
	assert.Empty(t, list)
	require.NoError(t, repo.SaveTemplate(ctx, brokerID, lead.Template{Profile: lead.ProfileHome, Tone: lead.ToneClose, Body: "v1"}))
	require.NoError(t, repo.SaveTemplate(ctx, brokerID, lead.Template{Profile: lead.ProfileHome, Tone: lead.ToneClose, Body: "v2"}))
	list, err = repo.Templates(ctx, brokerID)
	require.NoError(t, err)
	assert.Equal(t, []lead.Template{{Profile: lead.ProfileHome, Tone: lead.ToneClose, Body: "v2", Custom: true}}, list, "salvar de novo troca o texto")
	assert.Error(t, repo.SaveTemplate(ctx, brokerID, lead.Template{Profile: "x", Tone: lead.ToneClose, Body: "v"}))
	require.NoError(t, repo.DeleteTemplate(ctx, brokerID, lead.ProfileHome, lead.ToneClose))
	list, err = repo.Templates(ctx, brokerID)
	require.NoError(t, err)
	assert.Empty(t, list)

	require.NoError(t, database.Close())
	_, err = repo.Update(ctx, l.ID, lead.UpdateRequest{Name: "X"}, nil)
	assert.Error(t, err)
	_, err = repo.SetArchived(ctx, l.ID, true)
	assert.Error(t, err)
	_, err = repo.Templates(ctx, brokerID)
	assert.Error(t, err)
	assert.Error(t, repo.DeleteTemplate(ctx, brokerID, lead.ProfileHome, lead.ToneClose))
}

// ADR-0028: mensagem enviada pela fila — evento, etapa e data do último envio, fora dos sinais e do feed.
func TestLeadRepository_MessageSent(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewLeadRepository(database)
	ctx := context.Background()

	var devID, userID, brokerID string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT id FROM developments WHERE slug = 'boa-vista-residencial'`).Scan(&devID))
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('X', 'fab@x.com', 'h', 'corretor') RETURNING id`).Scan(&userID))
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO brokers (user_id, first_name, last_name, phone, creci, code) VALUES ($1, 'Fabiano', 'Souza', '54999990000', '', 'FAB-0001') RETURNING id`,
		userID).Scan(&brokerID))
	l, err := repo.CreateByBroker(ctx, brokerID, "h1", "conv1", lead.NewClientRequest{DevelopmentID: devID, Name: "Maria", Phone: "54988887777"})
	require.NoError(t, err)
	assert.Nil(t, l.LastMessageAt)
	before := l.LastActivityAt

	got, err := repo.MessageSent(ctx, l.ID, []byte(`{"perfil":"moradia","tom":"formal"}`))
	require.NoError(t, err)
	assert.Equal(t, lead.StageWorking, got.Stage, "novo passa para em atendimento")
	require.NotNil(t, got.LastMessageAt)
	assert.Empty(t, got.Signals, "envio do corretor não é sinal do cliente")
	assert.Equal(t, before.Unix(), got.LastActivityAt.Unix(), "nem atividade do cliente")
	feed, err := repo.Activity(ctx, brokerID)
	require.NoError(t, err)
	assert.Empty(t, feed, "nem entra no feed")
	events, err := repo.Events(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, lead.EventMessageSent, events[len(events)-1].Kind, "mas fica na jornada da ficha")

	_, err = repo.SetStage(ctx, l.ID, lead.StageRequest{Stage: lead.StageProposal})
	require.NoError(t, err)
	got, err = repo.MessageSent(ctx, l.ID, []byte(`{}`))
	require.NoError(t, err)
	assert.Equal(t, lead.StageProposal, got.Stage, "quem está adiante não muda")
	list, err := repo.List(ctx, lead.Filter{BrokerID: brokerID})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.NotNil(t, list[0].LastMessageAt, "a lista traz o último envio")

	_, err = repo.MessageSent(ctx, "00000000-0000-0000-0000-000000000000", []byte(`{}`))
	assert.Error(t, err, "cliente inexistente")
}
