package development

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlugify(t *testing.T) {
	assert.Equal(t, "boa-vista-residencial", Slugify("  Boa Vista  Residencial "))
	assert.Equal(t, "edificio-sao-joao-2", Slugify("Edifício São João — 2"))
	assert.Equal(t, "aeiou-c", Slugify("àéíóú_ç"))
	assert.Equal(t, "aeiou", Slugify("âëïõü"))
	assert.Equal(t, "aeiou", Slugify("ãèìòù"))
	assert.Equal(t, "aeio", Slugify("äêîô-"))
	assert.Equal(t, "", Slugify("!!!"))
}

func TestRequest_Validate(t *testing.T) {
	blank := " "
	r, err := Request{
		Name: " Boa Vista ", State: "rs", DeveloperID: &blank, CoverImageURL: "/img/a.jpg", TourURL: "https://tour.example/x",
		Gallery:    []Image{{URL: " /img/b.jpg ", Caption: " Fachada "}},
		Highlights: []Highlight{{Label: " Área ", Value: " 54 m² "}},
	}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "boa-vista", r.Slug, "sem slug, sai do nome")
	assert.Equal(t, "RS", r.State)
	assert.Equal(t, StatusLaunch, r.Status, "situação padrão")
	assert.Nil(t, r.DeveloperID)
	assert.Equal(t, Image{URL: "/img/b.jpg", Caption: "Fachada"}, r.Gallery[0])
	assert.Equal(t, Highlight{Label: "Área", Value: "54 m²"}, r.Highlights[0])

	ok := func() Request {
		return Request{Name: "Boa Vista", Slug: "boa-vista", Status: StatusReady, PresentationURL: " /showroom/bv/index.html "}
	}
	r, err = ok().Validate()
	require.NoError(t, err)
	assert.Equal(t, "/showroom/bv/index.html", r.PresentationURL)
	cases := []struct {
		name   string
		change func(*Request)
		want   error
	}{
		{"sem nome", func(r *Request) { r.Name = "" }, ErrNameRequired},
		{"nome longo", func(r *Request) { r.Name = strings.Repeat("a", 121) }, ErrFieldTooLong},
		{"slug", func(r *Request) { r.Slug = "Boa Vista" }, ErrInvalidSlug},
		{"slug vazio do nome", func(r *Request) { r.Slug = ""; r.Name = "!!!" }, ErrInvalidSlug},
		{"situação", func(r *Request) { r.Status = "vendido" }, ErrInvalidStatus},
		{"uf", func(r *Request) { r.State = "RGS" }, ErrInvalidState},
		{"descrição", func(r *Request) { r.Description = strings.Repeat("a", 4001) }, ErrFieldTooLong},
		{"cidade", func(r *Request) { r.City = strings.Repeat("a", 121) }, ErrFieldTooLong},
		{"galeria grande", func(r *Request) { r.Gallery = make([]Image, 41) }, ErrTooManyItems},
		{"capa javascript", func(r *Request) { r.CoverImageURL = "javascript:alert(1)" }, ErrInvalidURL},
		{"apresentação externa", func(r *Request) { r.PresentationURL = "https://x.example/p.html" }, ErrInvalidURL},
		{"apresentação protocolo relativo", func(r *Request) { r.PresentationURL = "//x.example/p.html" }, ErrInvalidURL},
		{"tour http", func(r *Request) { r.TourURL = "http://x" }, ErrInvalidURL},
		{"protocolo relativo", func(r *Request) { r.TourURL = "//evil.example" }, ErrInvalidURL},
		{"url longa", func(r *Request) { r.TourURL = "/" + strings.Repeat("a", 500) }, ErrInvalidURL},
		{"imagem vazia", func(r *Request) { r.Gallery = []Image{{URL: ""}} }, ErrInvalidURL},
		{"legenda longa", func(r *Request) { r.Gallery = []Image{{URL: "/a", Caption: strings.Repeat("a", 121)}} }, ErrInvalidURL},
		{"destaque vazio", func(r *Request) { r.Highlights = []Highlight{{Label: "Área"}} }, ErrInvalidHighlight},
		{"destaque longo", func(r *Request) { r.Highlights = []Highlight{{Label: "a", Value: strings.Repeat("a", 121)}} }, ErrFieldTooLong},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := ok()
			tc.change(&r)
			_, err := r.Validate()
			assert.ErrorIs(t, err, tc.want)
		})
	}
}
