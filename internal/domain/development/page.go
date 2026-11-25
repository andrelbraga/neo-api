package development

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// PageConfig é a edição da página de apresentação pelo admin (ADR-0027, RN-02.6): seções ocultas, blocos
// desligados e textos trocados. Vale para os layouts A e B. Os textos são achados pelo texto original, dentro
// da seção: se a página mudar e o original não existir mais, a troca simplesmente não se aplica.
type PageConfig struct {
	Sections map[string]SectionConfig `json:"sections"`
	Blocks   map[string]bool          `json:"blocks"`
	Texts    []TextOverride           `json:"texts"`
}

// SectionConfig é o que o admin muda numa seção inteira.
type SectionConfig struct {
	Hidden bool `json:"hidden"`
}

// TextOverride troca, na seção, o texto From por To. Em To, quebra de linha vira quebra de linha na página.
type TextOverride struct {
	Section string `json:"section"`
	From    string `json:"from"`
	To      string `json:"to"`
}

const (
	maxPageSections = 80
	maxPageBlocks   = 40
	maxPageTexts    = 400
	maxPageKey      = 80
	maxPageText     = 4000
)

var pageKeyRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Validate limpa a configuração: descarta trocas que não mudam nada, junta trocas repetidas (vale a última) e
// tira seções sem mudança. Chaves só com letras, números, ponto, hífen e sublinhado.
func (c PageConfig) Validate() (PageConfig, error) {
	out := PageConfig{Sections: map[string]SectionConfig{}, Blocks: map[string]bool{}, Texts: []TextOverride{}}
	if len(c.Sections) > maxPageSections || len(c.Blocks) > maxPageBlocks || len(c.Texts) > maxPageTexts {
		return out, ErrInvalidPage
	}
	for k, s := range c.Sections {
		if !validPageKey(k) {
			return out, ErrInvalidPage
		}
		if s.Hidden {
			out.Sections[k] = s
		}
	}
	for k, on := range c.Blocks {
		if !validPageKey(k) {
			return out, ErrInvalidPage
		}
		if !on {
			out.Blocks[k] = false
		}
	}
	seen := map[string]int{}
	for _, t := range c.Texts {
		t.From, t.To = strings.TrimSpace(t.From), strings.TrimSpace(t.To)
		if !validPageKey(t.Section) || t.From == "" || utf8.RuneCountInString(t.From) > maxPageText || utf8.RuneCountInString(t.To) > maxPageText {
			return out, ErrInvalidPage
		}
		if t.To == t.From {
			continue
		}
		key := t.Section + "\x00" + t.From
		if i, ok := seen[key]; ok {
			out.Texts[i] = t
			continue
		}
		seen[key] = len(out.Texts)
		out.Texts = append(out.Texts, t)
	}
	return out, nil
}

func validPageKey(k string) bool {
	return len(k) <= maxPageKey && pageKeyRe.MatchString(k)
}
