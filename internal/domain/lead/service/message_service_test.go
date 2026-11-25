package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/lead"
)

// RN-03.18: o corretor edita nome, WhatsApp e perfil; o WhatsApp segue a regra do cadastro.
func TestUpdateClient(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newBroker(repo)
	l, err := s.Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: dev, Name: "Maria", Phone: "54999990000"})
	require.NoError(t, err)
	mine, err := s.Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: dev, Name: "Outro", Phone: "54911110000"})
	require.NoError(t, err)
	_, err = s.Create(ctx, "u-ana", lead.NewClientRequest{DevelopmentID: dev, Name: "Da Ana", Phone: "54922220000"})
	require.NoError(t, err)

	got, err := s.Update(ctx, "u-fab", l.ID, lead.UpdateRequest{Name: "Maria Souza", Phone: "(54) 98888-0000", Profile: lead.ProfileInvestment})
	require.NoError(t, err)
	assert.Equal(t, "Maria Souza", got.Name)
	assert.Equal(t, "54988880000", got.Phone)
	assert.Equal(t, lead.ProfileInvestment, got.Profile)
	assert.Equal(t, []string{`{"antes":{"nome":"Maria","whatsapp":"54999990000"}}`}, repo.edits[l.ID], "guarda o que era antes")

	_, err = s.Update(ctx, "u-fab", l.ID, lead.UpdateRequest{Name: "Maria Souza", Phone: "54988880000", Profile: lead.ProfileHome})
	require.NoError(t, err)
	assert.Len(t, repo.edits[l.ID], 1, "só o perfil mudou: sem evento de edição")

	_, err = s.Update(ctx, "u-fab", l.ID, lead.UpdateRequest{Name: "Maria", Phone: mine.Phone})
	assert.ErrorIs(t, err, lead.ErrAlreadyYours)
	_, err = s.Update(ctx, "u-fab", l.ID, lead.UpdateRequest{Name: "Maria", Phone: "54922220000"})
	assert.ErrorIs(t, err, lead.ErrPhoneTaken)
	_, err = s.Update(ctx, "u-fab", l.ID, lead.UpdateRequest{Name: "Maria"})
	assert.ErrorIs(t, err, lead.ErrInvalidPhone, "quem já tinha WhatsApp não fica sem")
	_, err = s.Update(ctx, "u-ana", l.ID, lead.UpdateRequest{Name: "Maria", Phone: "54988880000"})
	assert.ErrorIs(t, err, lead.ErrNotFound, "só o dono edita")
	_, err = s.Update(ctx, "u-fab", l.ID, lead.UpdateRequest{})
	assert.ErrorIs(t, err, lead.ErrInvalidClientName)
}

func TestUpdateClient_VisitorWithoutPhone(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	_, err := newPublic(repo).Visit(ctx, "boa-vista", "FAB-0001", "", "", "")
	require.NoError(t, err)
	got, err := newBroker(repo).Update(ctx, "u-fab", "l1", lead.UpdateRequest{Name: "Visitante da feira"})
	require.NoError(t, err, "o visitante sem WhatsApp pode ganhar só o nome")
	assert.Equal(t, "Visitante da feira", got.Name)

	repo.failOn = "by-phone"
	_, err = newBroker(repo).Update(ctx, "u-fab", "l1", lead.UpdateRequest{Name: "X", Phone: "54999990000"})
	assert.EqualError(t, err, "by-phone")
}

// RN-03.19: excluir é arquivar; nada se perde e dá para restaurar.
func TestArchiveAndRestore(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newBroker(repo)
	l, err := s.Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: dev, Name: "Maria", Phone: "54999990000"})
	require.NoError(t, err)

	got, err := s.Archive(ctx, "u-fab", l.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.ArchivedAt)
	got, err = s.Restore(ctx, "u-fab", l.ID)
	require.NoError(t, err)
	assert.Nil(t, got.ArchivedAt)
	assert.Contains(t, repo.leads, l.ID, "nada é apagado")

	_, err = s.Archive(ctx, "u-ana", l.ID)
	assert.ErrorIs(t, err, lead.ErrNotFound, "só o dono arquiva")
}

// RN-03.20: doze modelos, os do corretor onde ele mudou e o padrão no resto.
func TestTemplatesService(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newBroker(repo)

	list, err := s.Templates(ctx, "u-fab")
	require.NoError(t, err)
	assert.Len(t, list, 12)
	assert.Equal(t, lead.Template{Profile: lead.ProfileGeneral, Tone: lead.ToneFormal, Body: lead.DefaultTemplate(lead.ProfileGeneral, lead.ToneFormal)}, list[0])

	saved, err := s.SaveTemplate(ctx, "u-fab", lead.ProfileHome, lead.ToneClose, lead.TemplateRequest{Body: " Oi, {nome}! {link} "})
	require.NoError(t, err)
	assert.Equal(t, lead.Template{Profile: lead.ProfileHome, Tone: lead.ToneClose, Body: "Oi, {nome}! {link}", Custom: true}, saved)
	list, err = s.Templates(ctx, "u-fab")
	require.NoError(t, err)
	assert.Equal(t, saved, list[4], "moradia/próximo é o do corretor")
	others, err := s.Templates(ctx, "u-ana")
	require.NoError(t, err)
	assert.False(t, others[4].Custom, "o modelo é de cada corretor")

	reset, err := s.ResetTemplate(ctx, "u-fab", lead.ProfileHome, lead.ToneClose)
	require.NoError(t, err)
	assert.Equal(t, lead.DefaultTemplate(lead.ProfileHome, lead.ToneClose), reset.Body)
	assert.False(t, reset.Custom)
	assert.Empty(t, repo.templates)

	_, err = s.SaveTemplate(ctx, "u-fab", "geral", "gritado", lead.TemplateRequest{Body: "x"})
	assert.ErrorIs(t, err, lead.ErrTemplateKey)
	_, err = s.ResetTemplate(ctx, "u-fab", "nenhum", lead.ToneFormal)
	assert.ErrorIs(t, err, lead.ErrTemplateKey)
	_, err = s.SaveTemplate(ctx, "u-fab", lead.ProfileHome, lead.ToneFormal, lead.TemplateRequest{})
	assert.ErrorIs(t, err, lead.ErrInvalidTemplate)
	for _, call := range []func() error{
		func() error { _, err := s.Templates(ctx, "u-x"); return err },
		func() error {
			_, err := s.SaveTemplate(ctx, "u-x", lead.ProfileHome, lead.ToneFormal, lead.TemplateRequest{Body: "x"})
			return err
		},
		func() error { _, err := s.ResetTemplate(ctx, "u-x", lead.ProfileHome, lead.ToneFormal); return err },
	} {
		assert.ErrorIs(t, call(), lead.ErrNotBroker)
	}
	for _, op := range []string{"templates", "save-template", "delete-template"} {
		repo.failOn = op
		_, e1 := s.Templates(ctx, "u-fab")
		_, e2 := s.SaveTemplate(ctx, "u-fab", lead.ProfileHome, lead.ToneFormal, lead.TemplateRequest{Body: "x"})
		_, e3 := s.ResetTemplate(ctx, "u-fab", lead.ProfileHome, lead.ToneFormal)
		assert.True(t, e1 != nil || e2 != nil || e3 != nil, op)
	}
}

// RN-03.22: o envio confirmado na fila vira evento e tira o cliente de "novo"; quem está adiante não muda.
func TestMessageSent(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newBroker(repo)
	l, err := s.Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: dev, Name: "Maria", Phone: "54999990000"})
	require.NoError(t, err)
	require.Equal(t, lead.StageNew, l.Stage)

	got, err := s.MessageSent(ctx, "u-fab", l.ID, lead.MessageSentRequest{Profile: lead.ProfileHome, Tone: lead.ToneClose})
	require.NoError(t, err)
	assert.Equal(t, lead.StageWorking, got.Stage)
	assert.NotNil(t, got.LastMessageAt)
	assert.Equal(t, []string{`{"perfil":"moradia","tom":"proximo"}`}, repo.sent[l.ID])

	repo.leads[l.ID].Stage = lead.StageProposal
	got, err = s.MessageSent(ctx, "u-fab", l.ID, lead.MessageSentRequest{Profile: lead.ProfileGeneral, Tone: lead.ToneFormal})
	require.NoError(t, err)
	assert.Equal(t, lead.StageProposal, got.Stage, "reenviar é permitido e não volta a etapa")
	assert.Len(t, repo.sent[l.ID], 2)

	_, err = s.MessageSent(ctx, "u-fab", l.ID, lead.MessageSentRequest{Profile: "x", Tone: lead.ToneFormal})
	assert.ErrorIs(t, err, lead.ErrInvalidMessage)
	_, err = s.MessageSent(ctx, "u-ana", l.ID, lead.MessageSentRequest{Profile: lead.ProfileHome, Tone: lead.ToneClose})
	assert.ErrorIs(t, err, lead.ErrNotFound, "só o dono registra")

	repo.leads[l.ID].Phone = ""
	_, err = s.MessageSent(ctx, "u-fab", l.ID, lead.MessageSentRequest{Profile: lead.ProfileHome, Tone: lead.ToneClose})
	assert.ErrorIs(t, err, lead.ErrNoPhone)

	repo.leads[l.ID].Phone = "54999990000"
	repo.failOn = "message-sent"
	_, err = s.MessageSent(ctx, "u-fab", l.ID, lead.MessageSentRequest{Profile: lead.ProfileHome, Tone: lead.ToneClose})
	assert.EqualError(t, err, "message-sent")
}
