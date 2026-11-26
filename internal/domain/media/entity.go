// Package media são as fotos guardadas no bucket (ADR-0029): a foto do corretor, que aparece no cartão
// "Seu consultor" da apresentação, e as fotos dos veículos que o cliente anexa à proposta (RN-03.23, RN-01.11).
// O bucket é privado: a API entrega a foto do corretor a qualquer um e as dos veículos só ao dono do cliente
// e ao admin.
package media

import (
	"regexp"
	"strings"
)

const (
	// PublicPath é o caminho da foto do corretor na API (sem o /api). O app e a apresentação juntam com o
	// endereço da API.
	PublicPath = "/public/media/"
	// BrokerPrefix e LeadPrefix separam no bucket as fotos dos corretores e as dos clientes.
	BrokerPrefix = "brokers/"
	LeadPrefix   = "leads/"

	// O app e a apresentação reduzem a foto antes de enviar; o limite é para o que passar disso.
	MaxBrokerPhotoBytes  = 5 << 20
	MaxVehiclePhotoBytes = 8 << 20
	// MaxPhotosPerLead limita quantas fotos de veículo um cliente envia, somando todas as propostas.
	MaxPhotosPerLead = 30
)

// Photo é a foto de veículo recebida; o id vai na proposta (condition.fotos).
type Photo struct {
	ID          string `json:"id"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
}

// BrokerPhoto é a resposta do envio da foto do corretor.
type BrokerPhoto struct {
	PhotoPath string `json:"photo_path"`
}

// Stored é o arquivo de uma foto no bucket.
type Stored struct {
	Key         string
	ContentType string
}

// BrokerPhotoPath é o caminho público da foto do corretor; vazio sem foto.
func BrokerPhotoPath(key string) string {
	if key == "" {
		return ""
	}
	return PublicPath + key
}

var brokerFileRe = regexp.MustCompile(`^[0-9a-f]{32}\.(jpg|png|webp)$`)

// BrokerKey é a chave no bucket da foto do corretor pelo nome do arquivo no endereço; vazio se o nome não é
// de uma foto de corretor (evita ler outras pastas do bucket).
func BrokerKey(file string) string {
	file = strings.TrimSpace(file)
	if !brokerFileRe.MatchString(file) {
		return ""
	}
	return BrokerPrefix + file
}
