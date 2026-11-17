// Package development é o empreendimento: o conteúdo que o admin cadastra e que o showroom mostra
// no link de cada corretor. Só empreendimento publicado aparece para corretor e cliente (RN-02.3).
package development

import (
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxName        = 120
	maxText        = 4000
	maxURL         = 500
	maxGallery     = 40
	maxHighlights  = 20
	maxShortField  = 120
	StatusLaunch   = "lancamento"
	StatusBuilding = "em_obras"
	StatusReady    = "pronto"
)

var (
	Statuses = []string{StatusLaunch, StatusBuilding, StatusReady}
	slugRe   = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	stateRe  = regexp.MustCompile(`^[A-Z]{2}$`)
)

type Image struct {
	URL     string `json:"url"`
	Caption string `json:"caption"`
}

type Highlight struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Layouts da página de apresentação (ADR-0019).
const (
	LayoutA = "a"
	LayoutB = "b"
)

type Development struct {
	ID            string  `json:"id"`
	Slug          string  `json:"slug"`
	Name          string  `json:"name"`
	DeveloperID   *string `json:"developer_id"`
	DeveloperName string  `json:"developer_name"`
	City          string  `json:"city"`
	State         string  `json:"state"`
	Status        string  `json:"status"`
	Published     bool    `json:"published"`
	Headline      string  `json:"headline"`
	Description   string  `json:"description"`
	CoverImageURL string  `json:"cover_image_url"`
	TourURL       string  `json:"tour_url"`
	// PresentationURL é a página própria do empreendimento; vazia, o showroom usa o modelo padrão (ADR-0008).
	PresentationURL string `json:"presentation_url"`
	// DefaultLayout é o layout da apresentação quando o link não pede um (ADR-0019): "a" ou "b".
	DefaultLayout string      `json:"default_layout"`
	Gallery       []Image     `json:"gallery"`
	Highlights    []Highlight `json:"highlights"`
	LeadsCount    int         `json:"leads_count"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type Request struct {
	Slug            string      `json:"slug"`
	Name            string      `json:"name"`
	DeveloperID     *string     `json:"developer_id"`
	City            string      `json:"city"`
	State           string      `json:"state"`
	Status          string      `json:"status"`
	Published       bool        `json:"published"`
	Headline        string      `json:"headline"`
	Description     string      `json:"description"`
	CoverImageURL   string      `json:"cover_image_url"`
	TourURL         string      `json:"tour_url"`
	PresentationURL string      `json:"presentation_url"`
	DefaultLayout   string      `json:"default_layout"`
	Gallery         []Image     `json:"gallery"`
	Highlights      []Highlight `json:"highlights"`
}

// Validate normaliza e confere o cadastro. Sem slug, ele sai do nome. Endereços de imagem e do tour
// aceitam caminho do próprio app ("/...") ou https, para não abrir brecha de script no showroom.
func (r Request) Validate() (Request, error) {
	r.Name = strings.TrimSpace(r.Name)
	r.Slug = strings.TrimSpace(r.Slug)
	if r.Slug == "" {
		r.Slug = Slugify(r.Name)
	}
	r.City = strings.TrimSpace(r.City)
	r.State = strings.ToUpper(strings.TrimSpace(r.State))
	r.Status = strings.TrimSpace(r.Status)
	if r.Status == "" {
		r.Status = StatusLaunch
	}
	r.Headline = strings.TrimSpace(r.Headline)
	r.Description = strings.TrimSpace(r.Description)
	r.CoverImageURL = strings.TrimSpace(r.CoverImageURL)
	r.TourURL = strings.TrimSpace(r.TourURL)
	r.PresentationURL = strings.TrimSpace(r.PresentationURL)
	r.DefaultLayout = strings.ToLower(strings.TrimSpace(r.DefaultLayout))
	if r.DefaultLayout == "" {
		r.DefaultLayout = LayoutA
	}
	if r.DeveloperID != nil && strings.TrimSpace(*r.DeveloperID) == "" {
		r.DeveloperID = nil
	}
	switch {
	case r.Name == "":
		return r, ErrNameRequired
	case utf8.RuneCountInString(r.Name) > maxName:
		return r, ErrFieldTooLong
	case !slugRe.MatchString(r.Slug) || len(r.Slug) > maxName:
		return r, ErrInvalidSlug
	case !slices.Contains(Statuses, r.Status):
		return r, ErrInvalidStatus
	case r.State != "" && !stateRe.MatchString(r.State):
		return r, ErrInvalidState
	case utf8.RuneCountInString(r.Description) > maxText:
		return r, ErrFieldTooLong
	case len(r.Gallery) > maxGallery || len(r.Highlights) > maxHighlights:
		return r, ErrTooManyItems
	case r.DefaultLayout != LayoutA && r.DefaultLayout != LayoutB:
		return r, ErrInvalidLayout
	}
	for _, f := range []string{r.City, r.Headline} {
		if utf8.RuneCountInString(f) > maxShortField {
			return r, ErrFieldTooLong
		}
	}
	// A apresentação roda no domínio do app, com acesso ao token do cliente: só caminho próprio, nunca externo.
	if r.PresentationURL != "" && (!strings.HasPrefix(r.PresentationURL, "/") || !safeURL(r.PresentationURL)) {
		return r, ErrInvalidURL
	}
	for _, u := range []string{r.CoverImageURL, r.TourURL} {
		if !safeURL(u) {
			return r, ErrInvalidURL
		}
	}
	gallery := make([]Image, 0, len(r.Gallery))
	for _, img := range r.Gallery {
		img.URL, img.Caption = strings.TrimSpace(img.URL), strings.TrimSpace(img.Caption)
		if img.URL == "" || !safeURL(img.URL) || utf8.RuneCountInString(img.Caption) > maxShortField {
			return r, ErrInvalidURL
		}
		gallery = append(gallery, img)
	}
	r.Gallery = gallery
	highlights := make([]Highlight, 0, len(r.Highlights))
	for _, h := range r.Highlights {
		h.Label, h.Value = strings.TrimSpace(h.Label), strings.TrimSpace(h.Value)
		if h.Label == "" || h.Value == "" {
			return r, ErrInvalidHighlight
		}
		if utf8.RuneCountInString(h.Label) > maxShortField || utf8.RuneCountInString(h.Value) > maxShortField {
			return r, ErrFieldTooLong
		}
		highlights = append(highlights, h)
	}
	r.Highlights = highlights
	return r, nil
}

// Slugify gera o trecho do endereço a partir do nome: minúsculas, sem acento, palavras unidas por hífen.
func Slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		r = fold(r)
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			dash = false
		case unicode.IsSpace(r) || r == '-' || r == '_':
			if b.Len() > 0 && !dash {
				b.WriteRune('-')
				dash = true
			}
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

func fold(r rune) rune {
	switch r {
	case 'á', 'à', 'â', 'ã', 'ä':
		return 'a'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï':
		return 'i'
	case 'ó', 'ò', 'ô', 'õ', 'ö':
		return 'o'
	case 'ú', 'ù', 'û', 'ü':
		return 'u'
	case 'ç':
		return 'c'
	}
	return r
}

func safeURL(u string) bool {
	if u == "" {
		return true
	}
	if len(u) > maxURL {
		return false
	}
	return (strings.HasPrefix(u, "/") && !strings.HasPrefix(u, "//")) || strings.HasPrefix(u, "https://")
}
