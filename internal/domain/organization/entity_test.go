package organization

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequest_Validate(t *testing.T) {
	r, err := Request{Kind: " Imobiliaria ", Name: " Omni ", Creci: "j-123", Email: " Contato@Omni.com "}.Validate()
	require.NoError(t, err)
	assert.Equal(t, KindAgency, r.Kind)
	assert.Equal(t, "Omni", r.Name)
	assert.Equal(t, "J-123", r.Creci)
	assert.Equal(t, "contato@omni.com", r.Email)

	_, err = Request{Kind: KindDeveloper, Name: "Dwell"}.Validate()
	assert.NoError(t, err, "e-mail é opcional")
	for _, logo := range []string{" /logos/omni.png ", "https://cdn.example/omni.png"} {
		r, err = Request{Kind: KindAgency, Name: "Omni", LogoURL: logo}.Validate()
		require.NoError(t, err)
		assert.NotContains(t, r.LogoURL, " ")
	}

	cases := []struct {
		name string
		r    Request
		want error
	}{
		{"tipo", Request{Kind: "construtora", Name: "X"}, ErrInvalidKind},
		{"nome", Request{Kind: KindAgency, Name: " "}, ErrNameRequired},
		{"nome longo", Request{Kind: KindAgency, Name: strings.Repeat("a", 121)}, ErrFieldTooLong},
		{"campo longo", Request{Kind: KindAgency, Name: "X", Website: strings.Repeat("a", 201)}, ErrFieldTooLong},
		{"cidade longa", Request{Kind: KindAgency, Name: "X", City: strings.Repeat("a", 201)}, ErrFieldTooLong},
		{"email", Request{Kind: KindAgency, Name: "X", Email: "x"}, ErrInvalidEmail},
		{"logo javascript", Request{Kind: KindAgency, Name: "X", LogoURL: "javascript:alert(1)"}, ErrInvalidLogo},
		{"logo protocolo relativo", Request{Kind: KindAgency, Name: "X", LogoURL: "//x.example/l.png"}, ErrInvalidLogo},
	}
	for _, tc := range cases {
		_, err := tc.r.Validate()
		assert.ErrorIs(t, err, tc.want, tc.name)
	}
}
