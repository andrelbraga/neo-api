// Package organization é o cadastro das empresas no admin: incorporadoras (donas dos empreendimentos)
// e imobiliárias (onde o corretor trabalha). As duas têm o mesmo cadastro; o tipo separa as telas.
package organization

import (
	"net/mail"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	KindDeveloper = "incorporadora"
	KindAgency    = "imobiliaria"

	maxName  = 120
	maxField = 200
)

var Kinds = []string{KindDeveloper, KindAgency}

type Organization struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	Document  string    `json:"document"`
	Creci     string    `json:"creci"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Website   string    `json:"website"`
	LogoURL   string    `json:"logo_url"`
	City      string    `json:"city"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type Request struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Document string `json:"document"`
	Creci    string `json:"creci"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Website  string `json:"website"`
	LogoURL  string `json:"logo_url"`
	City     string `json:"city"`
	Active   bool   `json:"active"`
}

// Validate: tipo conhecido, nome obrigatório; os demais campos são opcionais, com e-mail válido quando houver.
func (r Request) Validate() (Request, error) {
	r.Kind = strings.ToLower(strings.TrimSpace(r.Kind))
	r.Name = strings.TrimSpace(r.Name)
	r.Document = strings.TrimSpace(r.Document)
	r.Creci = strings.ToUpper(strings.TrimSpace(r.Creci))
	r.Phone = strings.TrimSpace(r.Phone)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Website = strings.TrimSpace(r.Website)
	r.LogoURL = strings.TrimSpace(r.LogoURL)
	r.City = strings.TrimSpace(r.City)
	switch {
	case !ValidKind(r.Kind):
		return r, ErrInvalidKind
	case r.Name == "":
		return r, ErrNameRequired
	case utf8.RuneCountInString(r.Name) > maxName:
		return r, ErrFieldTooLong
	}
	// O logo aparece na apresentação do cliente: só caminho do próprio app ou https.
	if r.LogoURL != "" && !((strings.HasPrefix(r.LogoURL, "/") && !strings.HasPrefix(r.LogoURL, "//")) || strings.HasPrefix(r.LogoURL, "https://")) {
		return r, ErrInvalidLogo
	}
	for _, f := range []string{r.Document, r.Creci, r.Phone, r.Email, r.Website, r.LogoURL, r.City} {
		if utf8.RuneCountInString(f) > maxField {
			return r, ErrFieldTooLong
		}
	}
	if r.Email != "" {
		if addr, err := mail.ParseAddress(r.Email); err != nil || addr.Address != r.Email {
			return r, ErrInvalidEmail
		}
	}
	return r, nil
}

func ValidKind(kind string) bool {
	return slices.Contains(Kinds, kind)
}
