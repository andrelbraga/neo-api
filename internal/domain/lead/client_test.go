package lead

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"(54) 99999-0000":     "54999990000",
		"+55 54 99999-0000":   "54999990000",
		"55 54 3286-0000":     "5432860000",
		"55 99999-0000":       "55999990000", // DDD 55 (Santa Maria) sem o país fica como está
		"54 3286-0000":        "5432860000",
		" wa: 0054 999990000": "0054999990000",
	}
	for in, want := range cases {
		assert.Equal(t, want, NormalizePhone(in), in)
	}
}

func TestNewClientRequest_Validate(t *testing.T) {
	r, err := NewClientRequest{DevelopmentID: " d1 ", Name: " Maria   da  Silva ", Phone: "+55 (54) 99999-0000"}.Validate()
	require.NoError(t, err)
	assert.Equal(t, NewClientRequest{DevelopmentID: "d1", Name: "Maria da Silva", Phone: "54999990000"}, r)

	_, err = NewClientRequest{Name: "Maria", Phone: "54999990000"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidDevelopment)
	_, err = NewClientRequest{DevelopmentID: "d1", Name: "  ", Phone: "54999990000"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidClientName)
	_, err = NewClientRequest{DevelopmentID: "d1", Name: strings.Repeat("a", 81), Phone: "54999990000"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidClientName)
	_, err = NewClientRequest{DevelopmentID: "d1", Name: "Maria", Phone: "9999-0000"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidPhone)
}

func TestParseClients(t *testing.T) {
	t.Run("cabeçalho em outra ordem, ponto e vírgula e BOM", func(t *testing.T) {
		rows, err := ParseClients([]byte("\xef\xbb\xbfE-mail;Telefone;Nome completo\r\nx@y;(54) 99999-0000; Maria \r\n;;\r\nz@w;54988887777;João\r\n"))
		require.NoError(t, err)
		assert.Equal(t, []ClientRow{{Line: 2, Name: "Maria", Phone: "(54) 99999-0000"}, {Line: 4, Name: "João", Phone: "54988887777"}}, rows)
	})
	t.Run("sem cabeçalho e com vírgula", func(t *testing.T) {
		rows, err := ParseClients([]byte("Maria,54999990000\n\"Souza, João\",54988887777\nSó o nome"))
		require.NoError(t, err)
		assert.Equal(t, []ClientRow{{1, "Maria", "54999990000"}, {2, "Souza, João", "54988887777"}, {3, "Só o nome", ""}}, rows)
	})
	t.Run("tabulação e WhatsApp no título", func(t *testing.T) {
		rows, err := ParseClients([]byte("nome\twhatsapp\nAna\t54999990000\n"))
		require.NoError(t, err)
		assert.Equal(t, []ClientRow{{2, "Ana", "54999990000"}}, rows)
	})
	t.Run("Latin-1 do Excel", func(t *testing.T) {
		rows, err := ParseClients([]byte("nome;celular\nJo\xe3o;54999990000\n"))
		require.NoError(t, err)
		assert.Equal(t, "João", rows[0].Name)
	})
	t.Run("vazia, quebrada ou grande demais", func(t *testing.T) {
		_, err := ParseClients([]byte("nome;telefone\n"))
		assert.ErrorIs(t, err, ErrInvalidFile, "só o cabeçalho")
		_, err = ParseClients([]byte(""))
		assert.ErrorIs(t, err, ErrInvalidFile)
		_, err = ParseClients([]byte(strings.Repeat("Ana;54999990000\n", MaxImportRows+1)))
		assert.ErrorIs(t, err, ErrTooManyRows)
		_, err = ParseClients(make([]byte, MaxImportBytes+1))
		assert.ErrorIs(t, err, ErrTooManyRows)
	})
	t.Run("até o limite de linhas", func(t *testing.T) {
		rows, err := ParseClients([]byte(strings.Repeat("Ana;54999990000\n", MaxImportRows)))
		require.NoError(t, err)
		assert.Len(t, rows, MaxImportRows)
	})
}
