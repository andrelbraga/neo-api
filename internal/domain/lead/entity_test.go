package lead

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventRequest_Validate(t *testing.T) {
	for _, k := range []string{"acao", "tempo", "perfil"} {
		_, err := EventRequest{Kind: k}.Validate()
		assert.NoError(t, err, "eventos da página de apresentação: "+k)
	}
	r, err := EventRequest{Kind: " secao ", Data: json.RawMessage(`{"id":"galeria","segundos":12}`)}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "secao", r.Kind)

	for _, data := range []json.RawMessage{nil, json.RawMessage(`null`)} {
		r, err = EventRequest{Kind: "whatsapp", Data: data}.Validate()
		require.NoError(t, err)
		assert.JSONEq(t, `{}`, string(r.Data))
	}

	for name, req := range map[string]EventRequest{
		"tipo desconhecido":             {Kind: "compra"},
		"visita só pela API":            {Kind: EventVisit},
		"contato só pela API":           {Kind: EventContact},
		"proposta tem endpoint próprio": {Kind: EventProposal},
		"chat saiu":                     {Kind: "chat_aberto"},
		"dados não objeto":              {Kind: "secao", Data: json.RawMessage(`[1]`)},
		"dados grandes":                 {Kind: "secao", Data: json.RawMessage(`{"x":"` + strings.Repeat("a", 2048) + `"}`)},
	} {
		_, err := req.Validate()
		assert.ErrorIs(t, err, ErrInvalidEvent, name)
	}
}

func TestContactRequest_Validate(t *testing.T) {
	r, err := ContactRequest{Name: " Maria ", Phone: "(54) 99999-0000"}.Validate()
	require.NoError(t, err)
	assert.Equal(t, ContactRequest{Name: "Maria", Phone: "54999990000"}, r)

	_, err = ContactRequest{Name: "", Phone: "54999990000"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidName)
	_, err = ContactRequest{Name: strings.Repeat("a", 81), Phone: "54999990000"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidName)
	_, err = ContactRequest{Name: "Maria", Phone: "9999"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidPhone)
	_, err = ContactRequest{Name: "Maria", Phone: "55 54 99999 00001"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidPhone)
}

func TestProposalRequest_Validate(t *testing.T) {
	r, err := ProposalRequest{Name: " Maria ", Phone: "(54) 99999-0000", Summary: " entrada de 150 mil ", Note: " ok "}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "Maria", r.Name)
	assert.Equal(t, "54999990000", r.Phone)
	assert.Equal(t, "entrada de 150 mil", r.Summary)
	assert.Equal(t, "ok", r.Note)
	assert.JSONEq(t, `{}`, string(r.Condition), "sem números vale só com o resumo")

	r, err = ProposalRequest{Name: "M", Phone: "54999990000", Condition: json.RawMessage(`{"preco":370000}`)}.Validate()
	require.NoError(t, err, "só com os números também vale")
	assert.JSONEq(t, `{"preco":370000}`, string(r.Condition))

	_, err = ProposalRequest{Phone: "54999990000", Summary: "x"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidName)
	for name, req := range map[string]ProposalRequest{
		"vazia":            {Name: "M", Phone: "54999990000"},
		"resumo longo":     {Name: "M", Phone: "54999990000", Summary: strings.Repeat("a", 901)},
		"observação longa": {Name: "M", Phone: "54999990000", Summary: "x", Note: strings.Repeat("a", 401)},
		"números em lista": {Name: "M", Phone: "54999990000", Summary: "x", Condition: json.RawMessage(`[1]`)},
		"números grandes":  {Name: "M", Phone: "54999990000", Condition: json.RawMessage(`{"x":"` + strings.Repeat("a", 2048) + `"}`)},
	} {
		_, err := req.Validate()
		assert.ErrorIs(t, err, ErrInvalidProposal, name)
	}
}

func TestAdjustRequest_Validate(t *testing.T) {
	r, err := AdjustRequest{BasedOn: " p1 ", Note: " desconto ", Condition: json.RawMessage(`{"preco":360000}`)}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "p1", r.BasedOn)
	assert.Equal(t, "desconto", r.Note)
	for name, req := range map[string]AdjustRequest{
		"sem números":      {},
		"sem preço":        {Condition: json.RawMessage(`{"entrada":1}`)},
		"preço em texto":   {Condition: json.RawMessage(`{"preco":"x"}`)},
		"observação longa": {Condition: json.RawMessage(`{"preco":1}`), Note: strings.Repeat("a", 401)},
	} {
		_, err := req.Validate()
		assert.ErrorIs(t, err, ErrInvalidProposal, name)
	}
}

func TestStageRequest_Validate(t *testing.T) {
	r, err := StageRequest{Stage: StageSold, SaleValue: 360000, ProposalID: " p1 ", Note: " assinado "}.Validate()
	require.NoError(t, err)
	assert.Equal(t, StageRequest{Stage: StageSold, SaleValue: 360000, ProposalID: "p1", Note: "assinado"}, r)

	r, err = StageRequest{Stage: StageLost, SaleValue: 1, ProposalID: "p1", Note: "comprou outro"}.Validate()
	require.NoError(t, err)
	assert.Equal(t, StageRequest{Stage: StageLost, Note: "comprou outro"}, r, "sem venda guarda só o motivo")

	r, err = StageRequest{Stage: StageWorking, SaleValue: 1, ProposalID: "p1", Note: "x"}.Validate()
	require.NoError(t, err)
	assert.Equal(t, StageRequest{Stage: StageWorking}, r, "etapa em aberto limpa o desfecho")

	_, err = StageRequest{Stage: "vendido"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidStage)
	_, err = StageRequest{Stage: StageSold}.Validate()
	assert.ErrorIs(t, err, ErrInvalidSale, "venda sem valor")
	_, err = StageRequest{Stage: StageLost, Note: strings.Repeat("a", 401)}.Validate()
	assert.ErrorIs(t, err, ErrInvalidSale)
}

func TestValidStage(t *testing.T) {
	for _, s := range Stages {
		assert.True(t, ValidStage(s), s)
	}
	assert.False(t, ValidStage("vendido"))
}
