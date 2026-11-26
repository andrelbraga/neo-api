package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"neo360-api/internal/config"
)

func TestNew_SemConfiguracaoFicaDesligado(t *testing.T) {
	b, err := New(config.Storage{})
	assert.NoError(t, err)
	assert.Nil(t, b)
}

func TestNew_EndpointInvalido(t *testing.T) {
	_, err := New(config.Storage{Endpoint: "sem-esquema", Bucket: "b", AccessKeyID: "k", SecretAccessKey: "s"})
	assert.Error(t, err)
}

func TestNew_Configurado(t *testing.T) {
	b, err := New(config.Storage{Endpoint: "https://t3.storageapi.dev", Bucket: "neo-files-x", AccessKeyID: "k", SecretAccessKey: "s", Region: "auto"})
	assert.NoError(t, err)
	assert.Equal(t, "neo-files-x", b.name)
}

func TestImage(t *testing.T) {
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F', 0}
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}
	webp := []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")
	for _, tc := range []struct {
		data      []byte
		ct, ext   string
		wantValid bool
	}{
		{jpeg, "image/jpeg", "jpg", true},
		{png, "image/png", "png", true},
		{webp, "image/webp", "webp", true},
		{[]byte("<svg xmlns='http://www.w3.org/2000/svg'></svg>"), "", "", false},
		{[]byte("GIF89a......"), "", "", false},
		{[]byte("%PDF-1.4"), "", "", false},
	} {
		ct, ext, ok := Image(tc.data)
		assert.Equal(t, tc.wantValid, ok, string(tc.data[:4]))
		if tc.wantValid {
			assert.Equal(t, tc.ct, ct)
			assert.Equal(t, tc.ext, ext)
		}
	}
}
