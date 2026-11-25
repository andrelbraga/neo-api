package lead

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

// Cliente cadastrado pelo corretor, um a um ou por planilha (RN-03.16). O corretor declara ter autorização
// para contatar os clientes que cadastra. O dono é quem cadastrou o WhatsApp primeiro no empreendimento: um
// WhatsApp que já é cliente de outro corretor é recusado, sem dizer de quem é.

const (
	MaxImportRows  = 500
	MaxImportBytes = 1 << 20

	// Situação de cada linha da planilha.
	RowOK          = "ok"
	RowInvalidName = "nome_invalido"
	RowInvalidPh   = "telefone_invalido"
	RowRepeated    = "repetido_no_arquivo"
	RowYours       = "ja_e_seu"
	RowTaken       = "ja_cadastrado"
)

// NewClientRequest é o cliente que o corretor cadastra à mão.
type NewClientRequest struct {
	DevelopmentID string `json:"development_id"`
	Name          string `json:"name"`
	Phone         string `json:"phone"`
}

// ClientRow é uma linha da planilha, com o número da linha no arquivo.
type ClientRow struct {
	Line  int    `json:"line"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// ImportRow é o resultado de uma linha: a situação e, se o cliente foi criado, o id dele.
type ImportRow struct {
	ClientRow
	Status string `json:"status"`
	LeadID string `json:"lead_id,omitempty"`
}

// ImportReport é a conferência (DryRun) ou o resultado da importação. Valid conta as linhas que entram;
// Created, as que entraram de fato.
type ImportReport struct {
	DryRun  bool        `json:"dry_run"`
	Total   int         `json:"total"`
	Valid   int         `json:"valid"`
	Created int         `json:"created"`
	Rows    []ImportRow `json:"rows"`
}

// Validate limpa nome e WhatsApp. O WhatsApp sai só com DDD e número (sem o 55 do país), como o cliente
// costuma digitar na apresentação, para que o mesmo número seja reconhecido como o mesmo cliente (RN-03.14).
func (r NewClientRequest) Validate() (NewClientRequest, error) {
	r.DevelopmentID = strings.TrimSpace(r.DevelopmentID)
	r.Name = strings.Join(strings.Fields(r.Name), " ")
	r.Phone = NormalizePhone(r.Phone)
	switch {
	case r.DevelopmentID == "":
		return r, ErrInvalidDevelopment
	case r.Name == "" || utf8.RuneCountInString(r.Name) > maxName:
		return r, ErrInvalidClientName
	case len(r.Phone) < minPhoneDigits || len(r.Phone) > maxPhoneDigits:
		return r, ErrInvalidPhone
	}
	return r, nil
}

// NormalizePhone deixa só os dígitos e tira o 55 do país de números com 12 ou 13 dígitos (55 + DDD + número).
func NormalizePhone(s string) string {
	d := digits(s)
	if (len(d) == 12 || len(d) == 13) && strings.HasPrefix(d, "55") {
		return d[2:]
	}
	return d
}

// ParseClients lê a planilha CSV de clientes: separador ponto e vírgula, vírgula ou tabulação; UTF-8 (com
// ou sem BOM) ou Latin-1, como o Excel salva. Com cabeçalho, as colunas são achadas pelo nome (nome;
// telefone, whatsapp ou celular); sem cabeçalho, a primeira coluna é o nome e a segunda o telefone. Linhas
// vazias são ignoradas.
func ParseClients(data []byte) ([]ClientRow, error) {
	if len(data) > MaxImportBytes {
		return nil, ErrTooManyRows
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		data = latin1(data)
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = separator(data)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	nameCol, phoneCol := 0, 1
	rows := []ClientRow{}
	first := true
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrInvalidFile
		}
		line, _ := r.FieldPos(0)
		if blank(rec) {
			continue
		}
		if first {
			first = false
			if n, p, ok := header(rec); ok {
				nameCol, phoneCol = n, p
				continue
			}
		}
		if len(rows) == MaxImportRows {
			return nil, ErrTooManyRows
		}
		rows = append(rows, ClientRow{Line: line, Name: field(rec, nameCol), Phone: field(rec, phoneCol)})
	}
	if len(rows) == 0 {
		return nil, ErrInvalidFile
	}
	return rows, nil
}

// header acha as colunas de nome e telefone pelo título. Sem as duas, a linha não é cabeçalho.
func header(rec []string) (nameCol, phoneCol int, ok bool) {
	nameCol, phoneCol = -1, -1
	for i, f := range rec {
		f = strings.ToLower(strings.TrimSpace(f))
		switch {
		case nameCol < 0 && strings.Contains(f, "nome"):
			nameCol = i
		case phoneCol < 0 && (strings.Contains(f, "telefone") || strings.Contains(f, "whatsapp") || strings.Contains(f, "celular") || strings.Contains(f, "fone")):
			phoneCol = i
		}
	}
	return nameCol, phoneCol, nameCol >= 0 && phoneCol >= 0
}

// separator escolhe, pela primeira linha, o separador que mais aparece; empate fica com o ponto e vírgula,
// o padrão do Excel em português.
func separator(data []byte) rune {
	line, _, _ := bytes.Cut(data, []byte("\n"))
	best, count := ';', bytes.Count(line, []byte(";"))
	for _, c := range []rune{',', '\t'} {
		if n := bytes.Count(line, []byte(string(c))); n > count {
			best, count = c, n
		}
	}
	return best
}

func latin1(b []byte) []byte {
	var out strings.Builder
	for _, c := range b {
		out.WriteRune(rune(c))
	}
	return []byte(out.String())
}

func blank(rec []string) bool {
	for _, f := range rec {
		if strings.TrimSpace(f) != "" {
			return false
		}
	}
	return true
}

func field(rec []string, i int) string {
	if i < len(rec) {
		return strings.TrimSpace(rec[i])
	}
	return ""
}
