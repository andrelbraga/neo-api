// Package broker é o corretor: a pessoa por trás da conta de papel "corretor". Ele se cadastra sozinho
// (RN-01.1) ou é cadastrado pelo admin, e o código dele identifica o link exclusivo (RN-02.1).
package broker

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxName        = 80
	maxCreci       = 20
	minPassword    = 8
	minPhoneDigits = 10
	maxPhoneDigits = 13
)

type Broker struct {
	ID         string  `json:"id"`
	UserID     string  `json:"user_id"`
	FirstName  string  `json:"first_name"`
	LastName   string  `json:"last_name"`
	Email      string  `json:"email"`
	Phone      string  `json:"phone"`
	Creci      string  `json:"creci"`
	Code       string  `json:"code"`
	AgencyID   *string `json:"agency_id"`
	AgencyName string  `json:"agency_name"`
	AgencyLogo string  `json:"agency_logo_url"`
	// PhotoPath é a foto do corretor na API (RN-01.11); vazio sem foto.
	PhotoPath  string    `json:"photo_path"`
	Active     bool      `json:"active"`
	LeadsCount int       `json:"leads_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// FullName é o nome que aparece para o cliente e nas listas.
func (b Broker) FullName() string {
	return strings.TrimSpace(b.FirstName + " " + b.LastName)
}

// Profile são os dados que o próprio corretor mantém (RN-01.2).
type Profile struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Creci     string `json:"creci"`
}

// CreateRequest é o cadastro aberto do corretor e o cadastro feito pelo admin.
type CreateRequest struct {
	Profile
	Email    string  `json:"email"`
	Password string  `json:"password"`
	AgencyID *string `json:"agency_id"`
}

// UpdateRequest é a edição pelo admin: perfil, e-mail de acesso, imobiliária e situação da conta.
// E-mail vazio mantém o atual.
type UpdateRequest struct {
	Profile
	Email    string  `json:"email"`
	AgencyID *string `json:"agency_id"`
	Active   bool    `json:"active"`
}

// Validate normaliza o perfil: nomes obrigatórios, telefone só com dígitos (com DDD), CRECI opcional.
func (p Profile) Validate() (Profile, error) {
	p.FirstName = strings.TrimSpace(p.FirstName)
	p.LastName = strings.TrimSpace(p.LastName)
	p.Creci = strings.ToUpper(strings.TrimSpace(p.Creci))
	p.Phone = Digits(p.Phone)
	switch {
	case p.FirstName == "" || p.LastName == "":
		return p, ErrNameRequired
	case utf8.RuneCountInString(p.FirstName) > maxName || utf8.RuneCountInString(p.LastName) > maxName:
		return p, ErrNameTooLong
	case len(p.Phone) < minPhoneDigits || len(p.Phone) > maxPhoneDigits:
		return p, ErrInvalidPhone
	case utf8.RuneCountInString(p.Creci) > maxCreci:
		return p, ErrInvalidCreci
	}
	return p, nil
}

func (r CreateRequest) Validate() (CreateRequest, error) {
	p, err := r.Profile.Validate()
	if err != nil {
		return r, err
	}
	r.Profile = p
	if r.Email, err = validEmail(r.Email); err != nil {
		return r, err
	}
	if utf8.RuneCountInString(r.Password) < minPassword {
		return r, ErrWeakPassword
	}
	r.AgencyID = blankToNil(r.AgencyID)
	return r, nil
}

func (r UpdateRequest) Validate() (UpdateRequest, error) {
	p, err := r.Profile.Validate()
	if err != nil {
		return r, err
	}
	r.Profile = p
	if strings.TrimSpace(r.Email) != "" {
		if r.Email, err = validEmail(r.Email); err != nil {
			return r, err
		}
	}
	r.AgencyID = blankToNil(r.AgencyID)
	return r, nil
}

// validEmail normaliza o e-mail (sem espaços, minúsculo) e confere o formato.
func validEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if addr, err := mail.ParseAddress(s); err != nil || addr.Address != s {
		return s, ErrInvalidEmail
	}
	return s, nil
}

// Digits deixa só os dígitos do telefone.
func Digits(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
}

// NewCode monta o código do link: três letras do primeiro nome e quatro dígitos (como FAB-0427 do protótipo).
func NewCode(firstName string, n int) string {
	letters := strings.Map(func(r rune) rune {
		r = unicode.ToUpper(foldAccent(r))
		if r >= 'A' && r <= 'Z' {
			return r
		}
		return -1
	}, firstName)
	letters += "XXX"
	return fmt.Sprintf("%s-%04d", letters[:3], n%10000)
}

func foldAccent(r rune) rune {
	switch unicode.ToLower(r) {
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

func blankToNil(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}
