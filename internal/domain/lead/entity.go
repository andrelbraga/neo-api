// Package lead é o cliente que abriu o link exclusivo de um corretor: o showroom público, o que ele
// fez na página (eventos), as propostas (do cliente e os ajustes do corretor) e o funil até o desfecho.
// A conversa acontece pelo WhatsApp; o chat da página saiu (ADR-0015).
//
// O dono do cliente é o corretor do link da primeira visita e não muda (RN-03.1). O navegador do
// cliente guarda só um token; o banco guarda o hash dele (RN-03.2).
package lead

import (
	"encoding/json"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"neo360-api/internal/domain/development"
)

const (
	StageNew      = "novo"
	StageWorking  = "em_atendimento"
	StageProposal = "proposta"
	StageSold     = "fechado"   // venda fechada (com valor)
	StageLost     = "encerrado" // encerrado sem venda

	AuthorClient = "cliente"
	AuthorBroker = "corretor"

	// EventVisit é registrado pela API a cada abertura do link; os demais vêm da página.
	EventVisit    = "visita"
	EventContact  = "contato"
	EventProposal = "proposta"

	// OriginLink é o cliente que chegou pelo link; OriginBroker, o que o corretor cadastrou (RN-03.16).
	OriginLink   = "link"
	OriginBroker = "corretor"

	maxEventData   = 2048
	maxSummary     = 900
	maxNote        = 400
	maxName        = 80
	minPhoneDigits = 10
	maxPhoneDigits = 13
)

var (
	// Stages é a ordem do funil (RN-03.6): três etapas em aberto e dois desfechos.
	Stages = []string{StageNew, StageWorking, StageProposal, StageSold, StageLost}
	// PageEvents são os eventos que a página pode registrar (RN-03.3). "acao" e "tempo" vêm da página de
	// apresentação do protótipo (cliques e segundos por seção); "perfil" é o perfil de cliente. A proposta
	// tem endpoint próprio (RN-03.8).
	PageEvents = []string{"secao", "galeria", "tour_3d", "whatsapp", "acao", "tempo", "perfil"}
)

// PublicBroker é o que o cliente vê do corretor no showroom.
type PublicBroker struct {
	ID         string `json:"-"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Creci      string `json:"creci"`
	Code       string `json:"code"`
	AgencyName string `json:"agency_name"`
	AgencyLogo string `json:"agency_logo_url"`
}

type Showroom struct {
	Development development.Development `json:"development"`
	Broker      PublicBroker            `json:"broker"`
}

// Visit é a resposta da abertura do link: o token que o navegador guarda e o contato, se já informado.
type Visit struct {
	Token string `json:"token"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type Lead struct {
	ID              string `json:"id"`
	DevelopmentID   string `json:"development_id"`
	DevelopmentName string `json:"development_name"`
	DevelopmentSlug string `json:"development_slug"`
	BrokerID        string `json:"broker_id"`
	BrokerName      string `json:"broker_name"`
	BrokerCode      string `json:"broker_code"`
	Name            string `json:"name"`
	Phone           string `json:"phone"`
	Stage           string `json:"stage"`
	EventsCount     int    `json:"events_count"`
	// NewProposals são as propostas do cliente que o corretor ainda não abriu.
	NewProposals   int       `json:"new_proposals"`
	LastActivityAt time.Time `json:"last_activity_at"`
	// Signals são os tipos de evento que o cliente já gerou (tour, proposta, WhatsApp...), para a lista.
	Signals   []string  `json:"signals"`
	CreatedAt time.Time `json:"created_at"`
	// Desfecho: data, valor e proposta de base da venda, ou o motivo de encerrar sem venda.
	OutcomeAt      *time.Time `json:"outcome_at"`
	SaleValue      *float64   `json:"sale_value"`
	SaleProposalID *string    `json:"sale_proposal_id"`
	OutcomeNote    string     `json:"outcome_note"`
	// Layout é o layout da apresentação na primeira visita (ADR-0019): "a" ou "b".
	Layout string `json:"layout"`
	// Origin diz se o cliente chegou pelo link ("link") ou foi cadastrado pelo corretor ("corretor", RN-03.16).
	Origin string `json:"origin"`
	// InviteCode é o código do link de convite do cliente cadastrado pelo corretor; vazio para quem veio pelo link.
	InviteCode string `json:"invite_code"`
	// Profile é o perfil de compra (RN-03.17): moradia, investimento, veraneio ou vazio (sem perfil).
	Profile string `json:"profile"`
	// ArchivedAt é quando o corretor arquivou o cliente (RN-03.19); nulo nos ativos.
	ArchivedAt *time.Time `json:"archived_at"`
	// LastMessageAt é o último envio de mensagem pelo WhatsApp que o corretor confirmou (RN-03.22).
	LastMessageAt *time.Time `json:"last_message_at"`
}

// LayoutStats é a conversão de um layout: clientes, quantos enviaram proposta e quantos fecharam.
type LayoutStats struct {
	Leads        int     `json:"leads"`
	WithProposal int     `json:"with_proposal"`
	Sales        int     `json:"sales"`
	SalesValue   float64 `json:"sales_value"`
}

// NormalizeLayout aceita só "b"; qualquer outro valor é o layout A, o padrão.
func NormalizeLayout(s string) string {
	if strings.EqualFold(strings.TrimSpace(s), "b") {
		return "b"
	}
	return "a"
}

// Activity é um evento com o cliente e o empreendimento, para o feed "ao vivo" do painel.
type Activity struct {
	LeadID          string          `json:"lead_id"`
	LeadName        string          `json:"lead_name"`
	DevelopmentName string          `json:"development_name"`
	BrokerName      string          `json:"broker_name"`
	Kind            string          `json:"kind"`
	Data            json.RawMessage `json:"data"`
	CreatedAt       time.Time       `json:"created_at"`
}

type Event struct {
	ID        int64           `json:"id"`
	Kind      string          `json:"kind"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
}

// Proposal é uma condição de compra: a que o cliente montou na apresentação ou um ajuste do corretor.
// Condition é o objeto que a apresentação calcula (preço, entrada, parcelas, reforços, chaves, veículos).
type Proposal struct {
	ID            string          `json:"id"`
	Author        string          `json:"author"`
	BasedOn       *string         `json:"based_on"`
	Condition     json.RawMessage `json:"condition"`
	Summary       string          `json:"summary"`
	Note          string          `json:"note"`
	WantsContract bool            `json:"wants_contract"`
	// ClientName e ClientPhone são o nome e o WhatsApp com que o cliente enviou a proposta (ADR-0016).
	ClientName  string     `json:"client_name"`
	ClientPhone string     `json:"client_phone"`
	SeenAt      *time.Time `json:"seen_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Detail struct {
	Lead
	Events    []Event    `json:"events"`
	Proposals []Proposal `json:"proposals"`
}

// Filter é a busca de clientes. BrokerID vazio só é aceito no admin.
type Filter struct {
	BrokerID      string
	DevelopmentID string
	Stage         string
	// Archived lista os arquivados no lugar dos ativos (RN-03.19).
	Archived bool
}

type Summary struct {
	Leads        int            `json:"leads"`
	Identified   int            `json:"identified"`
	NewProposals int            `json:"new_proposals"`
	Sales        int            `json:"sales"`
	SalesValue   float64        `json:"sales_value"`
	ByStage      map[string]int `json:"by_stage"`
	// ByLayout compara os layouts A e B da apresentação (ADR-0019).
	ByLayout map[string]LayoutStats `json:"by_layout"`
}

type EventRequest struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type ContactRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// ProposalRequest é a proposta enviada pelo cliente: vem com nome e WhatsApp para o corretor retornar.
type ProposalRequest struct {
	Name          string          `json:"name"`
	Phone         string          `json:"phone"`
	Summary       string          `json:"summary"`
	Note          string          `json:"note"`
	WantsContract bool            `json:"wants_contract"`
	Condition     json.RawMessage `json:"condition"`
}

// AdjustRequest é o ajuste do corretor sobre uma proposta (RN-03.12).
type AdjustRequest struct {
	BasedOn   string          `json:"based_on"`
	Summary   string          `json:"summary"`
	Note      string          `json:"note"`
	Condition json.RawMessage `json:"condition"`
}

// StageRequest muda a etapa. Venda fechada exige o valor; a proposta de base e a observação são opcionais.
// Em "sem venda", a observação é o motivo (RN-03.13).
type StageRequest struct {
	Stage      string  `json:"stage"`
	SaleValue  float64 `json:"sale_value"`
	ProposalID string  `json:"proposal_id"`
	Note       string  `json:"note"`
}

func (r EventRequest) Validate() (EventRequest, error) {
	r.Kind = strings.TrimSpace(r.Kind)
	if !slices.Contains(PageEvents, r.Kind) {
		return r, ErrInvalidEvent
	}
	if len(r.Data) == 0 || string(r.Data) == "null" {
		r.Data = json.RawMessage(`{}`)
	}
	var obj map[string]any
	if len(r.Data) > maxEventData || json.Unmarshal(r.Data, &obj) != nil {
		return r, ErrInvalidEvent
	}
	return r, nil
}

func (r ContactRequest) Validate() (ContactRequest, error) {
	r.Name = strings.TrimSpace(r.Name)
	r.Phone = digits(r.Phone)
	switch {
	case r.Name == "" || utf8.RuneCountInString(r.Name) > maxName:
		return r, ErrInvalidName
	case len(r.Phone) < minPhoneDigits || len(r.Phone) > maxPhoneDigits:
		return r, ErrInvalidPhone
	}
	return r, nil
}

func (r ProposalRequest) Validate() (ProposalRequest, error) {
	c, err := ContactRequest{Name: r.Name, Phone: r.Phone}.Validate()
	if err != nil {
		return r, err
	}
	r.Name, r.Phone = c.Name, c.Phone
	r.Summary, r.Note = strings.TrimSpace(r.Summary), strings.TrimSpace(r.Note)
	if utf8.RuneCountInString(r.Summary) > maxSummary || utf8.RuneCountInString(r.Note) > maxNote {
		return r, ErrInvalidProposal
	}
	// Sem os números (apresentação antiga ou sem simulador), a proposta vale só com o resumo.
	if len(r.Condition) == 0 || string(r.Condition) == "null" {
		r.Condition = json.RawMessage(`{}`)
	}
	if !jsonObject(r.Condition) || (r.Summary == "" && string(r.Condition) == "{}") {
		return r, ErrInvalidProposal
	}
	return r, nil
}

func (r AdjustRequest) Validate() (AdjustRequest, error) {
	r.BasedOn = strings.TrimSpace(r.BasedOn)
	r.Summary, r.Note = strings.TrimSpace(r.Summary), strings.TrimSpace(r.Note)
	if utf8.RuneCountInString(r.Summary) > maxSummary || utf8.RuneCountInString(r.Note) > maxNote || !jsonObject(r.Condition) {
		return r, ErrInvalidProposal
	}
	// O ajuste precisa ter preço: é o que o corretor está oferecendo.
	var c struct {
		Preco float64 `json:"preco"`
	}
	if json.Unmarshal(r.Condition, &c) != nil || c.Preco <= 0 {
		return r, ErrInvalidProposal
	}
	return r, nil
}

func (r StageRequest) Validate() (StageRequest, error) {
	r.Note, r.ProposalID = strings.TrimSpace(r.Note), strings.TrimSpace(r.ProposalID)
	switch {
	case !ValidStage(r.Stage):
		return r, ErrInvalidStage
	case utf8.RuneCountInString(r.Note) > maxNote:
		return r, ErrInvalidSale
	case r.Stage == StageSold && r.SaleValue <= 0:
		return r, ErrInvalidSale
	}
	if r.Stage != StageSold {
		r.SaleValue, r.ProposalID = 0, ""
	}
	if r.Stage != StageSold && r.Stage != StageLost {
		r.Note = ""
	}
	return r, nil
}

func ValidStage(s string) bool {
	return slices.Contains(Stages, s)
}

// jsonObject diz se o dado é um objeto JSON dentro do limite de tamanho.
func jsonObject(raw json.RawMessage) bool {
	var obj map[string]any
	return len(raw) > 0 && len(raw) <= maxEventData && json.Unmarshal(raw, &obj) == nil && obj != nil
}

func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
