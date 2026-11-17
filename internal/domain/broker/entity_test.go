package broker

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validCreate() CreateRequest {
	return CreateRequest{
		Profile:  Profile{FirstName: " Fabiano ", LastName: "Souza", Phone: "(54) 99999-0000", Creci: " 12345-f "},
		Email:    " Fabiano@Exemplo.com ",
		Password: "senha-longa",
	}
}

func TestCreateRequest_Validate(t *testing.T) {
	r, err := validCreate().Validate()
	require.NoError(t, err)
	assert.Equal(t, "Fabiano", r.FirstName)
	assert.Equal(t, "54999990000", r.Phone)
	assert.Equal(t, "12345-F", r.Creci)
	assert.Equal(t, "fabiano@exemplo.com", r.Email)
	assert.Nil(t, r.AgencyID)

	noCreci := validCreate()
	noCreci.Creci = ""
	_, err = noCreci.Validate()
	assert.NoError(t, err, "CRECI é opcional (RN-01.2)")

	blank := " "
	withBlankAgency := validCreate()
	withBlankAgency.AgencyID = &blank
	r, err = withBlankAgency.Validate()
	require.NoError(t, err)
	assert.Nil(t, r.AgencyID)

	cases := []struct {
		name   string
		change func(*CreateRequest)
		want   error
	}{
		{"sem nome", func(r *CreateRequest) { r.FirstName = " " }, ErrNameRequired},
		{"sem sobrenome", func(r *CreateRequest) { r.LastName = "" }, ErrNameRequired},
		{"nome longo", func(r *CreateRequest) { r.FirstName = strings.Repeat("a", 81) }, ErrNameTooLong},
		{"telefone curto", func(r *CreateRequest) { r.Phone = "9999-0000" }, ErrInvalidPhone},
		{"telefone longo", func(r *CreateRequest) { r.Phone = "55 54 99999 00001" }, ErrInvalidPhone},
		{"creci longo", func(r *CreateRequest) { r.Creci = strings.Repeat("1", 21) }, ErrInvalidCreci},
		{"email inválido", func(r *CreateRequest) { r.Email = "fabiano" }, ErrInvalidEmail},
		{"email com nome", func(r *CreateRequest) { r.Email = "Fab <fab@x.com>" }, ErrInvalidEmail},
		{"senha curta", func(r *CreateRequest) { r.Password = "1234567" }, ErrWeakPassword},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := validCreate()
			tc.change(&r)
			_, err := r.Validate()
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestUpdateRequest_Validate(t *testing.T) {
	agency := " a1 "
	r, err := UpdateRequest{Profile: Profile{FirstName: "Ana", LastName: "Lima", Phone: "54 99999 0000"}, AgencyID: &agency, Active: true}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "a1", *r.AgencyID)
	_, err = UpdateRequest{Profile: Profile{FirstName: "Ana"}}.Validate()
	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestNewCode(t *testing.T) {
	assert.Equal(t, "FAB-0427", NewCode("Fabiano", 427))
	assert.Equal(t, "JOA-0001", NewCode("João", 1))
	assert.Equal(t, "CEC-9999", NewCode("Çécília", 19999))
	assert.Equal(t, "IIA-0010", NewCode("Ìíâ", 10), "acentos viram letras simples")
	assert.Equal(t, "OUX-0005", NewCode("ôü", 5), "nome curto completa com X")
	assert.Equal(t, "XXX-0005", NewCode("123", 5))
	assert.Equal(t, "EEE-0000", NewCode("èëê", 0))
	assert.Equal(t, "AAA-0000", NewCode("àäã", 0))
	assert.Equal(t, "IOU-0000", NewCode("ïòù", 0))
	assert.Equal(t, "OUU-0000", NewCode("õûú", 0))
}

func TestFullNameAndDigits(t *testing.T) {
	assert.Equal(t, "Ana Lima", Broker{FirstName: "Ana", LastName: "Lima"}.FullName())
	assert.Equal(t, "Ana", Broker{FirstName: "Ana"}.FullName())
	assert.Equal(t, "5554999990000", Digits("+55 (54) 99999-0000"))
}
