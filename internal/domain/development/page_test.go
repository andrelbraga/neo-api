package development

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageConfig_Validate(t *testing.T) {
	c, err := PageConfig{
		Sections: map[string]SectionConfig{"h-planta": {Hidden: true}, "foot": {Hidden: false}},
		Blocks:   map[string]bool{"3d.condominio": false, "3d.apartamento": true},
		Texts: []TextOverride{
			{Section: "h-planta", From: " Uma planta ", To: "Sua planta"},
			{Section: "h-planta", From: "Uma planta", To: "Sua planta, do seu jeito"},
			{Section: "foot", From: "Igual", To: "Igual"},
			{Section: "foot", From: "Protótipo", To: ""},
		},
	}.Validate()
	require.NoError(t, err)
	assert.Equal(t, map[string]SectionConfig{"h-planta": {Hidden: true}}, c.Sections, "só o que muda")
	assert.Equal(t, map[string]bool{"3d.condominio": false}, c.Blocks, "só o que está desligado")
	assert.Equal(t, []TextOverride{
		{Section: "h-planta", From: "Uma planta", To: "Sua planta, do seu jeito"},
		{Section: "foot", From: "Protótipo", To: ""},
	}, c.Texts, "a última troca vale; troca igual sai; texto vazio apaga")

	empty, err := PageConfig{}.Validate()
	require.NoError(t, err)
	assert.NotNil(t, empty.Sections)
	assert.NotNil(t, empty.Texts)

	bad := []PageConfig{
		{Sections: map[string]SectionConfig{"a b": {Hidden: true}}},
		{Blocks: map[string]bool{strings.Repeat("a", 81): false}},
		{Texts: []TextOverride{{Section: "x", From: ""}}},
		{Texts: []TextOverride{{Section: "", From: "a"}}},
		{Texts: []TextOverride{{Section: "x", From: "a", To: strings.Repeat("b", 4001)}}},
		{Texts: make([]TextOverride, 401)},
	}
	for _, b := range bad {
		_, err := b.Validate()
		assert.ErrorIs(t, err, ErrInvalidPage)
	}
}
