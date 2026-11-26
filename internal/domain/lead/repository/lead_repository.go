package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"neo360-api/internal/domain/lead"
	"neo360-api/internal/domain/media"
	"neo360-api/internal/pkg/pgerr"
)

const (
	columns = `l.id, l.development_id, d.name, d.slug, l.broker_id, b.first_name || ' ' || b.last_name, b.code,
		l.name, l.phone, l.stage,
		(SELECT count(*) FROM lead_events e WHERE e.lead_id = l.id),
		(SELECT count(*) FROM proposals p WHERE p.lead_id = l.id AND p.author = 'cliente' AND p.seen_at IS NULL),
		(SELECT COALESCE(string_agg(DISTINCT e.kind, ','), '') FROM lead_events e WHERE e.lead_id = l.id AND e.kind <> 'mensagem_enviada'),
		l.last_activity_at, l.created_at, l.outcome_at, l.sale_value, l.sale_proposal_id, l.outcome_note, l.layout,
		l.origin, COALESCE(l.invite_code, ''), l.profile, l.archived_at,
		(SELECT max(e.created_at) FROM lead_events e WHERE e.lead_id = l.id AND e.kind = 'mensagem_enviada')`
	from = ` FROM leads l JOIN developments d ON d.id = l.development_id JOIN brokers b ON b.id = l.broker_id`

	queryShowroomBroker = `
		SELECT b.id, b.first_name || ' ' || b.last_name, b.phone, b.creci, b.code, COALESCE(o.name, ''), COALESCE(o.logo_url, ''), b.photo_key
		FROM brokers b JOIN users u ON u.id = b.user_id LEFT JOIN organizations o ON o.id = b.agency_id
		WHERE b.code = $1 AND u.active`
	queryBrokerByUser = `SELECT id FROM brokers WHERE user_id = $1`
	// Token de um cadastro juntado a outro (ADR-0016) abre o cadastro que ficou.
	queryByToken = `SELECT ` + columns + from + ` WHERE l.id = (SELECT COALESCE(merged_into, id) FROM leads WHERE token_hash = $1)`
	queryGet     = `SELECT ` + columns + from + ` WHERE l.id = $1 AND l.merged_into IS NULL`
	// O limite protege a tela de uma lista sem fim; os filtros por etapa e empreendimento reduzem o resto.
	queryList = `SELECT ` + columns + from + `
		WHERE l.merged_into IS NULL AND ($1 = '' OR l.broker_id::text = $1) AND ($2 = '' OR l.development_id::text = $2) AND ($3 = '' OR l.stage = $3)
			AND (l.archived_at IS NOT NULL) = $4
		ORDER BY l.last_activity_at DESC LIMIT 500`
	queryCreate = `INSERT INTO leads (development_id, broker_id, token_hash, layout) VALUES ($1, $2, $3, $4) RETURNING id`
	// Eventos do cliente contam como atividade.
	queryAddEvent = `
		WITH e AS (INSERT INTO lead_events (lead_id, kind, data) VALUES ($1, $2, $3) RETURNING lead_id)
		UPDATE leads SET last_activity_at = now() WHERE id = (SELECT lead_id FROM e)`
	queryContact = `
		WITH u AS (
			UPDATE leads SET name = $2, phone = $3, last_activity_at = now(), updated_at = now() WHERE id = $1 RETURNING id
		)
		INSERT INTO lead_events (lead_id, kind, data) SELECT id, 'contato', $4 FROM u`
	// O desfecho só fica preenchido em venda fechada ou sem venda; voltar a uma etapa em aberto o limpa.
	querySetStage = `
		UPDATE leads SET stage = $2, updated_at = now(),
			outcome_at = CASE WHEN $2 IN ('fechado', 'encerrado') THEN now() END,
			sale_value = CASE WHEN $2 = 'fechado' THEN $3::numeric END,
			sale_proposal_id = CASE WHEN $2 = 'fechado' THEN NULLIF($4, '')::uuid END,
			outcome_note = CASE WHEN $2 IN ('fechado', 'encerrado') THEN $5 ELSE '' END
		WHERE id = $1`
	queryEvents      = `SELECT id, kind, data, created_at FROM lead_events WHERE lead_id = $1 ORDER BY created_at, id LIMIT 1000`
	proposalColumns  = `id, author, based_on, condition, summary, note, wants_contract, client_name, client_phone, seen_at, created_at`
	queryProposals   = `SELECT ` + proposalColumns + ` FROM proposals WHERE lead_id = $1 ORDER BY created_at DESC, id`
	queryAddProposal = `
		INSERT INTO proposals (lead_id, author, based_on, condition, summary, note, wants_contract, client_name, client_phone)
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9) RETURNING ` + proposalColumns
	// A proposta do cliente também é um evento (jornada, sinais e feed ao vivo) e conta como atividade.
	queryProposalEvent = `
		WITH e AS (INSERT INTO lead_events (lead_id, kind, data) VALUES ($1, 'proposta', $2) RETURNING lead_id)
		UPDATE leads SET last_activity_at = now() WHERE id = (SELECT lead_id FROM e)`
	// Fotos de veículo (RN-03.23): a proposta do cliente leva as dele que ainda não estão em outra proposta,
	// inclusive as de um cadastro juntado a ele (ADR-0016). Ids vêm conferidos (lead.PhotoIDs).
	queryAttachPhotos = `
		UPDATE lead_photos SET proposal_id = $1
		WHERE id::text = ANY (string_to_array($3, ',')) AND proposal_id IS NULL
			AND lead_id IN (SELECT id FROM leads WHERE id = $2 OR merged_into = $2)
		RETURNING id::text`
	querySetPhotos = `UPDATE proposals SET condition = jsonb_set(condition, '{fotos}', $2::jsonb) WHERE id = $1 RETURNING condition`
	// Proposta leva o cliente à etapa "proposta" se ele ainda estava antes dela; desfecho não é tocado.
	queryAdvance = `UPDATE leads SET stage = 'proposta', updated_at = now() WHERE id = $1 AND stage IN ('novo', 'em_atendimento')`
	// Proposta nova do cliente tira o cadastro do arquivo: o corretor não pode deixar de vê-la (RN-03.19).
	queryUnarchive = `UPDATE leads SET archived_at = NULL, updated_at = now() WHERE id = $1 AND archived_at IS NOT NULL`
	querySeen      = `UPDATE proposals SET seen_at = now() WHERE lead_id = $1 AND author = 'cliente' AND seen_at IS NULL`
	queryActivity  = `
		SELECT l.id, l.name, d.name, b.first_name || ' ' || b.last_name, e.kind, e.data, e.created_at
		FROM lead_events e JOIN leads l ON l.id = e.lead_id JOIN developments d ON d.id = l.development_id
		JOIN brokers b ON b.id = l.broker_id
		WHERE l.merged_into IS NULL AND l.archived_at IS NULL AND ($1 = '' OR l.broker_id::text = $1) AND e.kind NOT IN ('tempo', 'secao', 'mensagem_enviada')
		ORDER BY e.created_at DESC, e.id DESC LIMIT 30`
	querySummary = `
		SELECT count(*),
			count(*) FILTER (WHERE l.phone <> ''),
			(SELECT count(*) FROM proposals p JOIN leads x ON x.id = p.lead_id
				WHERE p.author = 'cliente' AND p.seen_at IS NULL AND x.archived_at IS NULL AND ($1 = '' OR x.broker_id::text = $1)),
			COALESCE(sum(l.sale_value) FILTER (WHERE l.stage = 'fechado'), 0),
			count(*) FILTER (WHERE l.stage = 'novo'),
			count(*) FILTER (WHERE l.stage = 'em_atendimento'),
			count(*) FILTER (WHERE l.stage = 'proposta'),
			count(*) FILTER (WHERE l.stage = 'fechado'),
			count(*) FILTER (WHERE l.stage = 'encerrado')
		FROM leads l WHERE l.merged_into IS NULL AND l.archived_at IS NULL AND ($1 = '' OR l.broker_id::text = $1)`
	// Conversão por layout (ADR-0019): clientes, com proposta do cliente e com venda fechada.
	queryByLayout = `
		SELECT l.layout, count(*),
			count(*) FILTER (WHERE EXISTS (SELECT 1 FROM proposals p WHERE p.lead_id = l.id AND p.author = 'cliente')),
			count(*) FILTER (WHERE l.stage = 'fechado'),
			COALESCE(sum(l.sale_value) FILTER (WHERE l.stage = 'fechado'), 0)
		FROM leads l WHERE l.merged_into IS NULL AND l.archived_at IS NULL AND ($1 = '' OR l.broker_id::text = $1)
			-- Cliente cadastrado pelo corretor só conta no layout depois de abrir o link (RN-03.16).
			AND (l.origin = 'link' OR EXISTS (SELECT 1 FROM lead_events e WHERE e.lead_id = l.id AND e.kind = 'visita'))
		GROUP BY l.layout`
	// Cadastros do mesmo WhatsApp no mesmo empreendimento, o mais antigo primeiro: ele fica, os outros juntam.
	queryDuplicates = `
		SELECT d.id FROM leads l JOIN leads d ON d.development_id = l.development_id AND d.phone = l.phone AND d.merged_into IS NULL
		WHERE l.id = $1 AND l.phone <> '' ORDER BY d.created_at, d.id`
	queryPublished = `SELECT published FROM developments WHERE id = $1`
	queryByPhone   = `SELECT ` + columns + from + ` WHERE l.development_id = $1 AND l.phone = $2 AND l.merged_into IS NULL ORDER BY l.created_at, l.id LIMIT 1`
	queryByInvite  = `SELECT ` + columns + from + ` WHERE l.development_id = $1 AND l.invite_code = $2 AND l.merged_into IS NULL`
	// O cliente cadastrado pelo corretor nasce em "novo", com nome e WhatsApp; o token não é entregue a ninguém
	// (o navegador ganha o seu na primeira visita pelo convite, AddAlias).
	queryCreateByBroker = `
		INSERT INTO leads (development_id, broker_id, token_hash, name, phone, origin, invite_code)
		VALUES ($1, $2, $3, $4, $5, 'corretor', $6) RETURNING id`
	// O token do navegador vira um cadastro juntado ao cliente: queryByToken já abre o que ficou. Na primeira
	// visita, o cliente guarda o layout em que chegou (ADR-0019).
	queryAddAlias = `
		WITH k AS (SELECT id, development_id, broker_id FROM leads WHERE id = $1),
		a AS (INSERT INTO leads (development_id, broker_id, token_hash, layout, merged_into) SELECT development_id, broker_id, $2, $3, id FROM k)
		UPDATE leads SET layout = $3, updated_at = now()
		WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM lead_events e WHERE e.lead_id = $1 AND e.kind = 'visita')`
	// Edição do corretor: o evento guarda nome e WhatsApp de antes (como na troca feita pelo cliente, RN-03.10).
	queryUpdate    = `UPDATE leads SET name = $2, phone = $3, profile = $4, updated_at = now() WHERE id = $1 AND merged_into IS NULL`
	queryEditEvent = `INSERT INTO lead_events (lead_id, kind, data) VALUES ($1, 'edicao', $2)`
	// A mensagem enviada é do corretor: não conta como atividade do cliente, não é sinal nem entra no feed.
	// Só tira de "novo"; quem já está adiante fica onde está (RN-03.22).
	queryMessageSent = `
		WITH e AS (INSERT INTO lead_events (lead_id, kind, data) VALUES ($1, 'mensagem_enviada', $2) RETURNING lead_id)
		UPDATE leads SET stage = 'em_atendimento', updated_at = now() WHERE id = (SELECT lead_id FROM e) AND stage = 'novo'`
	queryArchive      = `UPDATE leads SET archived_at = CASE WHEN $2 THEN COALESCE(archived_at, now()) END, updated_at = now() WHERE id = $1 AND merged_into IS NULL`
	queryTemplates    = `SELECT profile, tone, body FROM message_templates WHERE broker_id = $1`
	querySaveTemplate = `
		INSERT INTO message_templates (broker_id, profile, tone, body) VALUES ($1, $2, $3, $4)
		ON CONFLICT (broker_id, profile, tone) DO UPDATE SET body = EXCLUDED.body, updated_at = now()`
	queryDeleteTemplate = `DELETE FROM message_templates WHERE broker_id = $1 AND profile = $2 AND tone = $3`
	queryMoveEvents     = `UPDATE lead_events SET lead_id = $1 WHERE lead_id = $2`
	queryMoveProposals  = `UPDATE proposals SET lead_id = $1 WHERE lead_id = $2`
	queryMarkMerged     = `UPDATE leads SET merged_into = $1, updated_at = now() WHERE id = $2 OR merged_into = $2`
	// O que ficou recebe o nome mais recente e a atividade mais recente dos juntados.
	queryTouchKept = `
		UPDATE leads k SET name = s.name, last_activity_at = GREATEST(k.last_activity_at, s.last_activity_at), updated_at = now()
		FROM leads s WHERE k.id = $1 AND s.id = $2`
)

type scanner interface {
	Scan(dest ...any) error
}

type LeadRepository struct {
	db *sql.DB
}

func NewLeadRepository(db *sql.DB) *LeadRepository {
	return &LeadRepository{db: db}
}

// ShowroomBroker acha o corretor ativo pelo código do link; corretor desativado derruba o link (RN-02.2).
func (r *LeadRepository) ShowroomBroker(ctx context.Context, code string) (lead.PublicBroker, error) {
	var b lead.PublicBroker
	var photo string
	err := r.db.QueryRowContext(ctx, queryShowroomBroker, code).Scan(&b.ID, &b.Name, &b.Phone, &b.Creci, &b.Code, &b.AgencyName, &b.AgencyLogo, &photo)
	b.PhotoPath = media.BrokerPhotoPath(photo)
	if errors.Is(err, sql.ErrNoRows) {
		return b, lead.ErrShowroomNotFound
	}
	if err != nil {
		return b, fmt.Errorf("showroom broker: %w", err)
	}
	return b, nil
}

func (r *LeadRepository) BrokerIDByUser(ctx context.Context, userID string) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, queryBrokerByUser, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) || pgerr.BadUUID(err) {
		return "", lead.ErrNotBroker
	}
	if err != nil {
		return "", fmt.Errorf("broker by user: %w", err)
	}
	return id, nil
}

func (r *LeadRepository) FindByToken(ctx context.Context, tokenHash string) (lead.Lead, error) {
	l, err := scan(r.db.QueryRowContext(ctx, queryByToken, tokenHash))
	return l, notFound(err, "find lead by token")
}

func (r *LeadRepository) Create(ctx context.Context, developmentID, brokerID, tokenHash, layout string) (lead.Lead, error) {
	var id string
	if err := r.db.QueryRowContext(ctx, queryCreate, developmentID, brokerID, tokenHash, layout).Scan(&id); err != nil {
		return lead.Lead{}, fmt.Errorf("create lead: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *LeadRepository) AddEvent(ctx context.Context, leadID, kind string, data []byte) error {
	if _, err := r.db.ExecContext(ctx, queryAddEvent, leadID, kind, data); err != nil {
		return fmt.Errorf("add event: %w", err)
	}
	return nil
}

// SetContact troca nome e WhatsApp e registra o evento de contato com eventData (os dados anteriores, se houver).
func (r *LeadRepository) SetContact(ctx context.Context, leadID string, c lead.ContactRequest, eventData []byte) error {
	if _, err := r.db.ExecContext(ctx, queryContact, leadID, c.Name, c.Phone, eventData); err != nil {
		return fmt.Errorf("set contact: %w", err)
	}
	return nil
}

func (r *LeadRepository) Get(ctx context.Context, id string) (lead.Lead, error) {
	l, err := scan(r.db.QueryRowContext(ctx, queryGet, id))
	return l, notFound(err, "get lead")
}

func (r *LeadRepository) List(ctx context.Context, f lead.Filter) ([]lead.Lead, error) {
	rows, err := r.db.QueryContext(ctx, queryList, f.BrokerID, f.DevelopmentID, f.Stage, f.Archived)
	if err != nil {
		return nil, fmt.Errorf("list leads: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	list := []lead.Lead{}
	for rows.Next() {
		l, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan lead: %w", err)
		}
		list = append(list, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list leads: %w", err)
	}
	return list, nil
}

func (r *LeadRepository) Events(ctx context.Context, leadID string) ([]lead.Event, error) {
	rows, err := r.db.QueryContext(ctx, queryEvents, leadID)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	list := []lead.Event{}
	for rows.Next() {
		var e lead.Event
		var data []byte
		if err := rows.Scan(&e.ID, &e.Kind, &data, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.Data = data
		list = append(list, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return list, nil
}

// SetStage muda a etapa e grava ou limpa o desfecho (RN-03.13).
func (r *LeadRepository) SetStage(ctx context.Context, id string, req lead.StageRequest) (lead.Lead, error) {
	res, err := r.db.ExecContext(ctx, querySetStage, id, req.Stage, req.SaleValue, req.ProposalID, req.Note)
	if err != nil {
		return lead.Lead{}, notFound(err, "set stage")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return lead.Lead{}, lead.ErrNotFound
	}
	return r.Get(ctx, id)
}

// Proposals devolve as propostas do cliente, da mais recente para a mais antiga.
func (r *LeadRepository) Proposals(ctx context.Context, leadID string) ([]lead.Proposal, error) {
	rows, err := r.db.QueryContext(ctx, queryProposals, leadID)
	if err != nil {
		return nil, fmt.Errorf("list proposals: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	list := []lead.Proposal{}
	for rows.Next() {
		p, err := scanProposal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan proposal: %w", err)
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list proposals: %w", err)
	}
	return list, nil
}

func (r *LeadRepository) AddProposal(ctx context.Context, leadID string, p lead.Proposal) (out lead.Proposal, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return out, fmt.Errorf("add proposal: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	basedOn := ""
	if p.BasedOn != nil {
		basedOn = *p.BasedOn
	}
	out, err = scanProposal(tx.QueryRowContext(ctx, queryAddProposal, leadID, p.Author, basedOn, []byte(p.Condition), p.Summary, p.Note, p.WantsContract, p.ClientName, p.ClientPhone))
	if err != nil {
		return out, notFound(err, "add proposal")
	}
	if p.Author == lead.AuthorClient {
		if out.Condition, err = attachPhotos(ctx, tx, out.ID, leadID, out.Condition); err != nil {
			return out, err
		}
		if _, err = tx.ExecContext(ctx, queryUnarchive, leadID); err != nil {
			return out, fmt.Errorf("unarchive: %w", err)
		}
		data, _ := json.Marshal(map[string]any{"proposal_id": out.ID, "resumo": p.Summary, "observacao": p.Note, "autorizou_contrato": p.WantsContract})
		if _, err = tx.ExecContext(ctx, queryProposalEvent, leadID, data); err != nil {
			return out, fmt.Errorf("proposal event: %w", err)
		}
	}
	if _, err = tx.ExecContext(ctx, queryAdvance, leadID); err != nil {
		return out, fmt.Errorf("advance stage: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return out, fmt.Errorf("add proposal: %w", err)
	}
	return out, nil
}

func (r *LeadRepository) MarkProposalsSeen(ctx context.Context, leadID string) error {
	if _, err := r.db.ExecContext(ctx, querySeen, leadID); err != nil {
		return fmt.Errorf("mark proposals seen: %w", err)
	}
	return nil
}

// MergeByPhone junta no cadastro mais antigo os outros do mesmo WhatsApp e empreendimento: eventos e
// propostas passam para ele, e os juntados apontam para ele (o token continua valendo). O dono é o do mais
// antigo, como na primeira visita (RN-03.1).
func (r *LeadRepository) MergeByPhone(ctx context.Context, leadID string) (kept string, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("merge by phone: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	rows, err := tx.QueryContext(ctx, queryDuplicates, leadID)
	if err != nil {
		return "", notFound(err, "merge by phone")
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			_ = rows.Close()
			return "", fmt.Errorf("merge by phone: %w", err)
		}
		ids = append(ids, id)
	}
	_ = rows.Close()
	if len(ids) < 2 {
		return leadID, tx.Commit()
	}
	kept = ids[0]
	for _, other := range ids[1:] {
		for _, q := range []string{queryTouchKept, queryMoveEvents, queryMoveProposals, queryMarkMerged} {
			if _, err = tx.ExecContext(ctx, q, kept, other); err != nil {
				return "", fmt.Errorf("merge by phone: %w", err)
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("merge by phone: %w", err)
	}
	return kept, nil
}

// Summary conta os clientes e as vendas de um corretor, ou de todos com brokerID vazio.
func (r *LeadRepository) Summary(ctx context.Context, brokerID string) (lead.Summary, error) {
	s := lead.Summary{ByStage: map[string]int{}}
	var n, w, p, sold, lost int
	err := r.db.QueryRowContext(ctx, querySummary, brokerID).Scan(&s.Leads, &s.Identified, &s.NewProposals, &s.SalesValue, &n, &w, &p, &sold, &lost)
	if err != nil {
		return s, fmt.Errorf("summary: %w", err)
	}
	s.Sales = sold
	s.ByStage[lead.StageNew], s.ByStage[lead.StageWorking], s.ByStage[lead.StageProposal] = n, w, p
	s.ByStage[lead.StageSold], s.ByStage[lead.StageLost] = sold, lost
	s.ByLayout = map[string]lead.LayoutStats{"a": {}, "b": {}}
	rows, err := r.db.QueryContext(ctx, queryByLayout, brokerID)
	if err != nil {
		return s, fmt.Errorf("summary by layout: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var layout string
		var st lead.LayoutStats
		if err := rows.Scan(&layout, &st.Leads, &st.WithProposal, &st.Sales, &st.SalesValue); err != nil {
			return s, fmt.Errorf("summary by layout: %w", err)
		}
		s.ByLayout[layout] = st
	}
	return s, rows.Err()
}

// Activity devolve os 30 eventos mais recentes de um corretor, ou de todos com brokerID vazio.
// Seções vistas e tempo por seção ficam fora: são volume, não novidade.
func (r *LeadRepository) Activity(ctx context.Context, brokerID string) ([]lead.Activity, error) {
	rows, err := r.db.QueryContext(ctx, queryActivity, brokerID)
	if err != nil {
		return nil, fmt.Errorf("activity: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	list := []lead.Activity{}
	for rows.Next() {
		var a lead.Activity
		var data []byte
		if err := rows.Scan(&a.LeadID, &a.LeadName, &a.DevelopmentName, &a.BrokerName, &a.Kind, &data, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		a.Data = data
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("activity: %w", err)
	}
	return list, nil
}

// DevelopmentPublished confere que o corretor cadastra o cliente num empreendimento que ele pode ofertar (RN-02.3).
func (r *LeadRepository) DevelopmentPublished(ctx context.Context, developmentID string) error {
	var published bool
	err := r.db.QueryRowContext(ctx, queryPublished, developmentID).Scan(&published)
	if errors.Is(err, sql.ErrNoRows) || pgerr.BadUUID(err) || (err == nil && !published) {
		return lead.ErrInvalidDevelopment
	}
	if err != nil {
		return fmt.Errorf("development published: %w", err)
	}
	return nil
}

func (r *LeadRepository) FindByPhone(ctx context.Context, developmentID, phone string) (lead.Lead, error) {
	l, err := scan(r.db.QueryRowContext(ctx, queryByPhone, developmentID, phone))
	return l, notFound(err, "find lead by phone")
}

func (r *LeadRepository) CreateByBroker(ctx context.Context, brokerID, tokenHash, inviteCode string, c lead.NewClientRequest) (lead.Lead, error) {
	var id string
	err := r.db.QueryRowContext(ctx, queryCreateByBroker, c.DevelopmentID, brokerID, tokenHash, c.Name, c.Phone, inviteCode).Scan(&id)
	if pgerr.BadUUID(err) || pgerr.ForeignKey(err) {
		return lead.Lead{}, lead.ErrInvalidDevelopment
	}
	if err != nil {
		return lead.Lead{}, fmt.Errorf("create lead by broker: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *LeadRepository) FindByInvite(ctx context.Context, developmentID, code string) (lead.Lead, error) {
	l, err := scan(r.db.QueryRowContext(ctx, queryByInvite, developmentID, code))
	return l, notFound(err, "find lead by invite")
}

func (r *LeadRepository) AddAlias(ctx context.Context, leadID, tokenHash, layout string) error {
	if _, err := r.db.ExecContext(ctx, queryAddAlias, leadID, tokenHash, layout); err != nil {
		return fmt.Errorf("add alias: %w", err)
	}
	return nil
}

// Update troca nome, WhatsApp e perfil e, com eventData, grava o evento de edição, na mesma transação.
func (r *LeadRepository) Update(ctx context.Context, id string, req lead.UpdateRequest, eventData []byte) (out lead.Lead, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return out, fmt.Errorf("update lead: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	res, err := tx.ExecContext(ctx, queryUpdate, id, req.Name, req.Phone, req.Profile)
	if err != nil {
		return out, notFound(err, "update lead")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return out, lead.ErrNotFound
	}
	if len(eventData) > 0 {
		if _, err = tx.ExecContext(ctx, queryEditEvent, id, eventData); err != nil {
			return out, fmt.Errorf("edit event: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return out, fmt.Errorf("update lead: %w", err)
	}
	return r.Get(ctx, id)
}

// SetArchived arquiva (mantendo a data do primeiro arquivamento) ou restaura o cliente.
func (r *LeadRepository) SetArchived(ctx context.Context, id string, archived bool) (lead.Lead, error) {
	res, err := r.db.ExecContext(ctx, queryArchive, id, archived)
	if err != nil {
		return lead.Lead{}, notFound(err, "archive lead")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return lead.Lead{}, lead.ErrNotFound
	}
	return r.Get(ctx, id)
}

func (r *LeadRepository) Templates(ctx context.Context, brokerID string) ([]lead.Template, error) {
	rows, err := r.db.QueryContext(ctx, queryTemplates, brokerID)
	if err != nil {
		return nil, fmt.Errorf("templates: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	list := []lead.Template{}
	for rows.Next() {
		t := lead.Template{Custom: true}
		if err := rows.Scan(&t.Profile, &t.Tone, &t.Body); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		list = append(list, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("templates: %w", err)
	}
	return list, nil
}

func (r *LeadRepository) SaveTemplate(ctx context.Context, brokerID string, t lead.Template) error {
	if _, err := r.db.ExecContext(ctx, querySaveTemplate, brokerID, t.Profile, t.Tone, t.Body); err != nil {
		return fmt.Errorf("save template: %w", err)
	}
	return nil
}

func (r *LeadRepository) DeleteTemplate(ctx context.Context, brokerID, profile, tone string) error {
	if _, err := r.db.ExecContext(ctx, queryDeleteTemplate, brokerID, profile, tone); err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	return nil
}

// MessageSent grava o envio e, se o cliente ainda estava em novo, o leva para em atendimento.
func (r *LeadRepository) MessageSent(ctx context.Context, id string, eventData []byte) (lead.Lead, error) {
	if _, err := r.db.ExecContext(ctx, queryMessageSent, id, eventData); err != nil {
		return lead.Lead{}, notFound(err, "message sent")
	}
	return r.Get(ctx, id)
}

func scan(s scanner) (lead.Lead, error) {
	var l lead.Lead
	var signals string
	var outcomeAt sql.NullTime
	var saleValue sql.NullFloat64
	var saleProposal sql.NullString
	var archivedAt, lastMessageAt sql.NullTime
	err := s.Scan(&l.ID, &l.DevelopmentID, &l.DevelopmentName, &l.DevelopmentSlug, &l.BrokerID, &l.BrokerName, &l.BrokerCode,
		&l.Name, &l.Phone, &l.Stage, &l.EventsCount, &l.NewProposals, &signals, &l.LastActivityAt, &l.CreatedAt,
		&outcomeAt, &saleValue, &saleProposal, &l.OutcomeNote, &l.Layout, &l.Origin, &l.InviteCode, &l.Profile, &archivedAt, &lastMessageAt)
	if archivedAt.Valid {
		l.ArchivedAt = &archivedAt.Time
	}
	if lastMessageAt.Valid {
		l.LastMessageAt = &lastMessageAt.Time
	}
	if outcomeAt.Valid {
		l.OutcomeAt = &outcomeAt.Time
	}
	if saleValue.Valid {
		l.SaleValue = &saleValue.Float64
	}
	if saleProposal.Valid {
		l.SaleProposalID = &saleProposal.String
	}
	l.Signals = []string{}
	if signals != "" {
		l.Signals = strings.Split(signals, ",")
	}
	return l, err
}

// attachPhotos liga à proposta as fotos de veículo que ela cita e deixa em condition.fotos só as que ficaram.
func attachPhotos(ctx context.Context, tx *sql.Tx, proposalID, leadID string, condition json.RawMessage) (json.RawMessage, error) {
	ids := lead.PhotoIDs(condition)
	if len(ids) == 0 && !hasPhotos(condition) {
		return condition, nil
	}
	rows, err := tx.QueryContext(ctx, queryAttachPhotos, proposalID, leadID, strings.Join(ids, ","))
	if err != nil {
		return condition, fmt.Errorf("attach photos: %w", err)
	}
	attached := map[string]bool{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			_ = rows.Close()
			return condition, fmt.Errorf("attach photos: %w", err)
		}
		attached[id] = true
	}
	if err = rows.Close(); err != nil {
		return condition, fmt.Errorf("attach photos: %w", err)
	}
	kept := []string{}
	for _, id := range ids {
		if attached[id] {
			kept = append(kept, id)
		}
	}
	list, _ := json.Marshal(kept)
	var out []byte
	if err = tx.QueryRowContext(ctx, querySetPhotos, proposalID, list).Scan(&out); err != nil {
		return condition, fmt.Errorf("set photos: %w", err)
	}
	return out, nil
}

// hasPhotos diz se a proposta trouxe o campo fotos, mesmo sem nenhum id válido (aí ele fica vazio).
func hasPhotos(condition json.RawMessage) bool {
	var c map[string]json.RawMessage
	if json.Unmarshal(condition, &c) != nil {
		return false
	}
	_, ok := c["fotos"]
	return ok
}

func scanProposal(s scanner) (lead.Proposal, error) {
	var p lead.Proposal
	var basedOn sql.NullString
	var seen sql.NullTime
	var cond []byte
	err := s.Scan(&p.ID, &p.Author, &basedOn, &cond, &p.Summary, &p.Note, &p.WantsContract, &p.ClientName, &p.ClientPhone, &seen, &p.CreatedAt)
	p.Condition = cond
	if basedOn.Valid {
		p.BasedOn = &basedOn.String
	}
	if seen.Valid {
		p.SeenAt = &seen.Time
	}
	return p, err
}

func notFound(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) || pgerr.BadUUID(err) {
		return lead.ErrNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
