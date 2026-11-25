package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"neo360-api/internal/domain/lead"
)

// Create cadastra um cliente à mão (RN-03.16). O dono é quem cadastrou o WhatsApp primeiro no empreendimento:
// se já é cliente de outro corretor, a resposta não diz de quem é.
func (s *BrokerService) Create(ctx context.Context, userID string, r lead.NewClientRequest) (lead.Lead, error) {
	valid, err := r.Validate()
	if err != nil {
		return lead.Lead{}, err
	}
	brokerID, err := s.repo.BrokerIDByUser(ctx, userID)
	if err != nil {
		return lead.Lead{}, err
	}
	if err := s.repo.DevelopmentPublished(ctx, valid.DevelopmentID); err != nil {
		return lead.Lead{}, err
	}
	switch status, err := s.check(ctx, brokerID, valid); {
	case err != nil:
		return lead.Lead{}, err
	case status == lead.RowYours:
		return lead.Lead{}, lead.ErrAlreadyYours
	case status == lead.RowTaken:
		return lead.Lead{}, lead.ErrPhoneTaken
	}
	return s.create(ctx, brokerID, valid)
}

// Import confere e cadastra os clientes da planilha. Cada linha tem a sua situação; só as válidas entram.
// Com dryRun, nada é gravado: é a conferência que o corretor vê antes de confirmar.
func (s *BrokerService) Import(ctx context.Context, userID, developmentID string, data []byte, dryRun bool) (lead.ImportReport, error) {
	brokerID, err := s.repo.BrokerIDByUser(ctx, userID)
	if err != nil {
		return lead.ImportReport{}, err
	}
	developmentID = strings.TrimSpace(developmentID)
	if developmentID == "" {
		return lead.ImportReport{}, lead.ErrInvalidDevelopment
	}
	if err := s.repo.DevelopmentPublished(ctx, developmentID); err != nil {
		return lead.ImportReport{}, err
	}
	rows, err := lead.ParseClients(data)
	if err != nil {
		return lead.ImportReport{}, err
	}
	report := lead.ImportReport{DryRun: dryRun, Total: len(rows), Rows: make([]lead.ImportRow, 0, len(rows))}
	inFile := map[string]bool{}
	for _, row := range rows {
		out := lead.ImportRow{ClientRow: row}
		valid, err := lead.NewClientRequest{DevelopmentID: developmentID, Name: row.Name, Phone: row.Phone}.Validate()
		switch {
		case errors.Is(err, lead.ErrInvalidClientName):
			out.Status = lead.RowInvalidName
		case err != nil:
			out.Status = lead.RowInvalidPh
		case inFile[valid.Phone]:
			out.Status = lead.RowRepeated
		default:
			inFile[valid.Phone] = true
			if out.Status, err = s.check(ctx, brokerID, valid); err != nil {
				return lead.ImportReport{}, err
			}
		}
		if err == nil {
			out.Name, out.Phone = valid.Name, valid.Phone
		}
		if out.Status == lead.RowOK {
			report.Valid++
			if !dryRun {
				l, err := s.create(ctx, brokerID, valid)
				if err != nil {
					return lead.ImportReport{}, err
				}
				out.LeadID = l.ID
				report.Created++
			}
		}
		report.Rows = append(report.Rows, out)
	}
	return report, nil
}

// check diz se o WhatsApp está livre no empreendimento, já é deste corretor ou já é de outro.
func (s *BrokerService) check(ctx context.Context, brokerID string, c lead.NewClientRequest) (string, error) {
	l, err := s.repo.FindByPhone(ctx, c.DevelopmentID, c.Phone)
	switch {
	case errors.Is(err, lead.ErrNotFound):
		return lead.RowOK, nil
	case err != nil:
		return "", err
	case l.BrokerID == brokerID:
		return lead.RowYours, nil
	}
	return lead.RowTaken, nil
}

func (s *BrokerService) create(ctx context.Context, brokerID string, c lead.NewClientRequest) (lead.Lead, error) {
	token, err := s.newToken()
	if err != nil {
		return lead.Lead{}, err
	}
	code, err := s.newCode()
	if err != nil {
		return lead.Lead{}, err
	}
	return s.repo.CreateByBroker(ctx, brokerID, hash(token), code, c)
}

// inviteCode é o código do link de convite: 10 letras e números minúsculos, sem confusão entre 0/o e 1/l.
func inviteCode() (string, error) {
	const alphabet = "abcdefghijkmnpqrstuvwxyz23456789"
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("invite code: %w", err)
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b), nil
}
