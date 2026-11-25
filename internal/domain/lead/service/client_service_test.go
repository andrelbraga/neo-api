package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/lead"
)

func newBroker(repo *memRepo) *BrokerService {
	s := NewBrokerService(repo)
	n := 0
	s.newToken = func() (string, error) { n++; return fmt.Sprintf("guardado%d", n), nil }
	s.newCode = func() (string, error) { return fmt.Sprintf("conv%d", n), nil }
	return s
}

const dev = "d-boa-vista"

// RN-03.16: o corretor cadastra o cliente à mão; ele nasce em "novo", com convite próprio.
func TestCreateClient(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newBroker(repo)

	l, err := s.Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: dev, Name: "  Maria   Souza ", Phone: "+55 (54) 99999-0000"})
	require.NoError(t, err)
	assert.Equal(t, "b-fab", l.BrokerID, "o dono é quem cadastrou")
	assert.Equal(t, "Maria Souza", l.Name)
	assert.Equal(t, "54999990000", l.Phone, "WhatsApp sem o 55 do país, como o cliente digita")
	assert.Equal(t, lead.StageNew, l.Stage)
	assert.Equal(t, lead.OriginBroker, l.Origin)
	assert.Equal(t, "conv1", l.InviteCode)
	assert.NotContains(t, repo.byHash, "guardado1", "o banco guarda só o hash do token")

	_, err = s.Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: dev, Name: "Maria", Phone: "54 99999-0000"})
	assert.ErrorIs(t, err, lead.ErrAlreadyYours)
	_, err = s.Create(ctx, "u-ana", lead.NewClientRequest{DevelopmentID: dev, Name: "Maria", Phone: "54999990000"})
	assert.ErrorIs(t, err, lead.ErrPhoneTaken, "WhatsApp de outro corretor: vale quem cadastrou primeiro")
	assert.NotContains(t, err.Error(), "Fabiano", "não diz de quem é o cliente")
}

func TestCreateClient_Errors(t *testing.T) {
	ctx := context.Background()
	ok := lead.NewClientRequest{DevelopmentID: dev, Name: "Maria", Phone: "54999990000"}
	_, err := newBroker(newMem()).Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: dev, Phone: "54999990000"})
	assert.ErrorIs(t, err, lead.ErrInvalidClientName)
	_, err = newBroker(newMem()).Create(ctx, "u-x", ok)
	assert.ErrorIs(t, err, lead.ErrNotBroker)
	_, err = newBroker(newMem()).Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: "outro", Name: "Maria", Phone: "54999990000"})
	assert.ErrorIs(t, err, lead.ErrInvalidDevelopment, "só empreendimento publicado")

	for _, op := range []string{"by-phone", "create-by-broker"} {
		repo := newMem()
		repo.failOn = op
		_, err := newBroker(repo).Create(ctx, "u-fab", ok)
		assert.EqualError(t, err, op)
	}
	s := newBroker(newMem())
	s.newToken = func() (string, error) { return "", errors.New("rand") }
	_, err = s.Create(ctx, "u-fab", ok)
	assert.EqualError(t, err, "rand")
	s = newBroker(newMem())
	s.newCode = func() (string, error) { return "", errors.New("code") }
	_, err = s.Create(ctx, "u-fab", ok)
	assert.EqualError(t, err, "code")
}

// RN-03.16: cada linha da planilha tem a sua situação; a conferência não grava nada.
func TestImport(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	s := newBroker(repo)
	_, err := s.Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: dev, Name: "Já Meu", Phone: "54911110000"})
	require.NoError(t, err)
	_, err = s.Create(ctx, "u-ana", lead.NewClientRequest{DevelopmentID: dev, Name: "Da Ana", Phone: "54922220000"})
	require.NoError(t, err)

	csv := "nome;telefone\nMaria;(54) 99999-0000\n;54999991111\nJoão;123\nMaria de novo;5554999990000\nJá Meu;54911110000\nDa Ana;54922220000\nPedro;54988887777\n"
	report, err := s.Import(ctx, "u-fab", dev, []byte(csv), true)
	require.NoError(t, err)
	assert.True(t, report.DryRun)
	assert.Equal(t, 7, report.Total)
	assert.Equal(t, 2, report.Valid)
	assert.Equal(t, 0, report.Created)
	assert.Len(t, repo.leads, 2, "a conferência não grava")
	status := []string{}
	for _, r := range report.Rows {
		status = append(status, r.Status)
	}
	assert.Equal(t, []string{lead.RowOK, lead.RowInvalidName, lead.RowInvalidPh, lead.RowRepeated, lead.RowYours, lead.RowTaken, lead.RowOK}, status)
	assert.Equal(t, 2, report.Rows[0].Line, "a linha conta o cabeçalho")
	assert.Equal(t, "54999990000", report.Rows[0].Phone, "a linha válida volta normalizada")
	assert.Equal(t, "123", report.Rows[2].Phone, "a inválida volta como veio")

	report, err = s.Import(ctx, "u-fab", dev, []byte(csv), false)
	require.NoError(t, err)
	assert.Equal(t, 2, report.Created)
	assert.Len(t, repo.leads, 4)
	assert.NotEmpty(t, report.Rows[0].LeadID)
	assert.Equal(t, "b-fab", repo.leads[report.Rows[0].LeadID].BrokerID)
	assert.Empty(t, report.Rows[1].LeadID)
}

func TestImport_Errors(t *testing.T) {
	ctx := context.Background()
	csv := []byte("Maria;54999990000")
	_, err := newBroker(newMem()).Import(ctx, "u-x", dev, csv, true)
	assert.ErrorIs(t, err, lead.ErrNotBroker)
	_, err = newBroker(newMem()).Import(ctx, "u-fab", " ", csv, true)
	assert.ErrorIs(t, err, lead.ErrInvalidDevelopment)
	repo := newMem()
	repo.unpublished = true
	_, err = newBroker(repo).Import(ctx, "u-fab", dev, csv, true)
	assert.ErrorIs(t, err, lead.ErrInvalidDevelopment)
	_, err = newBroker(newMem()).Import(ctx, "u-fab", dev, []byte("\n\n"), true)
	assert.ErrorIs(t, err, lead.ErrInvalidFile)

	for _, op := range []string{"by-phone", "create-by-broker"} {
		repo := newMem()
		repo.failOn = op
		_, err := newBroker(repo).Import(ctx, "u-fab", dev, csv, false)
		assert.EqualError(t, err, op)
	}
}

// RN-03.16: o link de convite leva a visita à ficha do cliente cadastrado.
func TestVisit_Invite(t *testing.T) {
	ctx := context.Background()
	repo := newMem()
	l, err := newBroker(repo).Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: dev, Name: "Maria", Phone: "54999990000"})
	require.NoError(t, err)
	s := newPublic(repo)

	v, err := s.Visit(ctx, "boa-vista", "FAB-0001", "", "b", l.InviteCode)
	require.NoError(t, err)
	assert.Equal(t, lead.Visit{Token: "tok1", Name: "Maria", Phone: "54999990000"}, v, "a página já sabe quem é")
	assert.Len(t, repo.leads, 1, "não nasce cliente novo")
	assert.Equal(t, l.ID, repo.aliases[hash("tok1")])
	assert.Equal(t, "b", repo.leads[l.ID].Layout, "a primeira visita grava o layout")
	assert.Equal(t, []string{`visita{"codigo":"FAB-0001"}`}, repo.events[l.ID])

	v, err = s.Visit(ctx, "boa-vista", "FAB-0001", "tok1", "a", l.InviteCode)
	require.NoError(t, err)
	assert.Equal(t, "tok1", v.Token, "o mesmo navegador continua com o mesmo token")
	assert.Equal(t, "b", repo.leads[l.ID].Layout, "volta pelo outro layout, mas conta no da primeira visita")
	assert.Len(t, repo.events[l.ID], 2)

	v, err = s.Visit(ctx, "boa-vista", "FAB-0001", "token-de-outro", "", l.InviteCode)
	require.NoError(t, err)
	assert.Equal(t, "tok2", v.Token, "outro navegador ganha um token ligado ao mesmo cliente")
	assert.Equal(t, l.ID, repo.aliases[hash("tok2")])

	v, err = s.Visit(ctx, "boa-vista", "ANA-0002", "", "", "nao-existe")
	require.NoError(t, err)
	assert.Equal(t, "b-ana", repo.leads["l2"].BrokerID, "convite que não existe vale como o link do corretor")
	assert.Equal(t, "tok3", v.Token)
}

func TestVisit_InviteErrors(t *testing.T) {
	ctx := context.Background()
	for _, op := range []string{"by-invite", "find", "alias", "event"} {
		repo := newMem()
		l, err := newBroker(repo).Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: dev, Name: "Maria", Phone: "54999990000"})
		require.NoError(t, err)
		repo.failOn = op
		_, err = newPublic(repo).Visit(ctx, "boa-vista", "FAB-0001", "algum", "", l.InviteCode)
		assert.EqualError(t, err, op)
	}
	repo := newMem()
	l, err := newBroker(repo).Create(ctx, "u-fab", lead.NewClientRequest{DevelopmentID: dev, Name: "Maria", Phone: "54999990000"})
	require.NoError(t, err)
	s := newPublic(repo)
	s.newToken = func() (string, error) { return "", errors.New("rand") }
	_, err = s.Visit(ctx, "boa-vista", "FAB-0001", "", "", l.InviteCode)
	assert.EqualError(t, err, "rand")
}

func TestInviteCode(t *testing.T) {
	a, err := inviteCode()
	require.NoError(t, err)
	b, _ := inviteCode()
	assert.Len(t, a, 10)
	assert.NotEqual(t, a, b)
	assert.Regexp(t, `^[a-km-np-z2-9]{10}$`, a)
}
