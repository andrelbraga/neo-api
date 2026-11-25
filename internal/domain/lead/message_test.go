package lead

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateRequest_Validate(t *testing.T) {
	r, err := UpdateRequest{Name: " Maria  Souza ", Phone: "+55 54 99999-0000", Profile: " moradia "}.Validate()
	require.NoError(t, err)
	assert.Equal(t, UpdateRequest{Name: "Maria Souza", Phone: "54999990000", Profile: "moradia"}, r)

	r, err = UpdateRequest{Name: "Maria"}.Validate()
	require.NoError(t, err, "WhatsApp vazio passa aqui; o serviço decide")
	assert.Equal(t, ProfileNone, r.Profile)

	_, err = UpdateRequest{Name: " "}.Validate()
	assert.ErrorIs(t, err, ErrInvalidClientName)
	_, err = UpdateRequest{Name: "Maria", Phone: "123"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidPhone)
	_, err = UpdateRequest{Name: "Maria", Profile: "geral"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidProfile, "geral é só do modelo, não da ficha")
}

func TestTemplates(t *testing.T) {
	for _, p := range TemplateProfiles {
		for _, tone := range Tones {
			body := DefaultTemplate(p, tone)
			assert.Contains(t, body, "{link}", p+"/"+tone)
			assert.Contains(t, body, "{nome}", p+"/"+tone)
			assert.True(t, ValidTemplateKey(p, tone))
		}
	}
	assert.False(t, ValidTemplateKey("", ToneFormal))
	assert.False(t, ValidTemplateKey(ProfileHome, "gritado"))

	r, err := TemplateRequest{Body: "  Olá, {nome}: {link} "}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "Olá, {nome}: {link}", r.Body)
	_, err = TemplateRequest{Body: "  "}.Validate()
	assert.ErrorIs(t, err, ErrInvalidTemplate)
	_, err = TemplateRequest{Body: strings.Repeat("a", 1001)}.Validate()
	assert.ErrorIs(t, err, ErrInvalidTemplate)
}

func TestMessageSentRequest_Validate(t *testing.T) {
	require.NoError(t, MessageSentRequest{Profile: ProfileGeneral, Tone: ToneDirect}.Validate())
	assert.ErrorIs(t, MessageSentRequest{Profile: ProfileHome}.Validate(), ErrInvalidMessage)
	assert.ErrorIs(t, MessageSentRequest{Tone: ToneFormal}.Validate(), ErrInvalidMessage)
}
