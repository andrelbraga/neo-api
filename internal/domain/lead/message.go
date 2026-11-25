package lead

import (
	"slices"
	"strings"
	"unicode/utf8"
)

// Perfil de compra, edição e arquivamento do cliente, e os modelos de mensagem do WhatsApp (ADR-0025).

const (
	ProfileNone       = ""
	ProfileHome       = "moradia"
	ProfileInvestment = "investimento"
	ProfileVacation   = "veraneio"
	// ProfileGeneral é o modelo de quem ainda não tem perfil.
	ProfileGeneral = "geral"

	ToneFormal = "formal"
	ToneClose  = "proximo"
	ToneDirect = "direto"

	// EventEdit é a troca de nome ou WhatsApp feita pelo corretor, com os dados de antes.
	EventEdit = "edicao"

	maxTemplate = 1000
)

var (
	// Profiles são os perfis que o corretor escolhe na ficha (RN-03.17); vazio é "sem perfil".
	Profiles = []string{ProfileNone, ProfileHome, ProfileInvestment, ProfileVacation}
	// TemplateProfiles são os perfis dos modelos: o geral vale para quem não tem perfil.
	TemplateProfiles = []string{ProfileGeneral, ProfileHome, ProfileInvestment, ProfileVacation}
	Tones            = []string{ToneFormal, ToneClose, ToneDirect}
)

// UpdateRequest é a edição do cliente pelo corretor (RN-03.18): nome, WhatsApp e perfil.
type UpdateRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Profile string `json:"profile"`
}

// Template é o texto de uma mensagem por perfil e tom. Custom diz se é do corretor ou o padrão.
// O texto leva {nome}, {empreendimento}, {link} e {corretor}, que a ficha preenche.
type Template struct {
	Profile string `json:"profile"`
	Tone    string `json:"tone"`
	Body    string `json:"body"`
	Custom  bool   `json:"custom"`
}

type TemplateRequest struct {
	Body string `json:"body"`
}

// Validate limpa os campos. O nome é obrigatório; o WhatsApp pode ficar vazio só em quem ainda não tinha
// (o visitante que não deixou contato), e quem decide isso é o serviço.
func (r UpdateRequest) Validate() (UpdateRequest, error) {
	r.Name = strings.Join(strings.Fields(r.Name), " ")
	r.Phone = NormalizePhone(r.Phone)
	r.Profile = strings.TrimSpace(r.Profile)
	switch {
	case r.Name == "" || utf8.RuneCountInString(r.Name) > maxName:
		return r, ErrInvalidClientName
	case r.Phone != "" && (len(r.Phone) < minPhoneDigits || len(r.Phone) > maxPhoneDigits):
		return r, ErrInvalidPhone
	case !slices.Contains(Profiles, r.Profile):
		return r, ErrInvalidProfile
	}
	return r, nil
}

func (r TemplateRequest) Validate() (TemplateRequest, error) {
	r.Body = strings.TrimSpace(r.Body)
	if r.Body == "" || utf8.RuneCountInString(r.Body) > maxTemplate {
		return r, ErrInvalidTemplate
	}
	return r, nil
}

// ValidTemplateKey diz se perfil e tom existem nos modelos.
func ValidTemplateKey(profile, tone string) bool {
	return slices.Contains(TemplateProfiles, profile) && slices.Contains(Tones, tone)
}

// DefaultTemplate é o texto padrão de um perfil e tom, usado enquanto o corretor não escreve o dele.
func DefaultTemplate(profile, tone string) string {
	return defaultTemplates[profile][tone]
}

// Textos padrão: rascunho de 24/11/2026, a confirmar pelo produto (P-19). O corretor ajusta os dele.
var defaultTemplates = map[string]map[string]string{
	ProfileGeneral: {
		ToneFormal: "Olá, {nome}. Aqui é {corretor}, corretor do {empreendimento}. Preparei uma apresentação com fotos, planta, condições e a maquete 3D para você conhecer com calma: {link}\n\nFico à disposição para qualquer dúvida.",
		ToneClose:  "Oi, {nome}! Tudo bem? Aqui é {corretor}. Separei o {empreendimento} para você dar uma olhada com calma: tem fotos, planta e até a maquete 3D. {link}\n\nMe conta o que achou!",
		ToneDirect: "{nome}, segue o {empreendimento}: fotos, planta, condições e maquete 3D em {link}\n\nPosso te ajudar a montar uma proposta?",
	},
	ProfileHome: {
		ToneFormal: "Olá, {nome}. Aqui é {corretor}. Pensando no seu novo lar, preparei a apresentação do {empreendimento}, com fotos, planta e condições de pagamento: {link}\n\nFico à disposição para qualquer dúvida.",
		ToneClose:  "Oi, {nome}! Aqui é {corretor}. Lembrei de você quando vi o {empreendimento}: dá para ver a planta, as fotos e o apartamento em 3D, como se já estivesse lá dentro. {link}\n\nMe fala o que achou!",
		ToneDirect: "{nome}, para morar: o {empreendimento}, com planta, fotos e condições em {link}\n\nQuer simular uma proposta?",
	},
	ProfileInvestment: {
		ToneFormal: "Olá, {nome}. Aqui é {corretor}. Como você busca investimento, preparei a apresentação do {empreendimento}, com as condições de pagamento e o simulador de proposta: {link}\n\nFico à disposição para conversarmos sobre os números.",
		ToneClose:  "Oi, {nome}! Aqui é {corretor}. Separei o {empreendimento} pensando no seu investimento: as condições estão na apresentação e dá para simular a proposta do seu jeito. {link}\n\nVamos conversar sobre os números?",
		ToneDirect: "{nome}, oportunidade de investimento: {empreendimento}, com condições e simulador em {link}\n\nMonto a proposta com você?",
	},
	ProfileVacation: {
		ToneFormal: "Olá, {nome}. Aqui é {corretor}. Para quem procura um lugar para as férias, preparei a apresentação do {empreendimento}, com fotos, planta e condições: {link}\n\nFico à disposição para qualquer dúvida.",
		ToneClose:  "Oi, {nome}! Aqui é {corretor}. Já imaginou ter o seu canto para as férias? Separei o {empreendimento} para você conhecer: fotos, planta e maquete 3D em {link}\n\nMe conta o que achou!",
		ToneDirect: "{nome}, para a sua casa de férias: {empreendimento}, com fotos, planta e condições em {link}\n\nQuer simular uma proposta?",
	},
}
