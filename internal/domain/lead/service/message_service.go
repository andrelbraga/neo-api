package service

import (
	"context"
	"encoding/json"
	"errors"

	"neo360-api/internal/domain/lead"
)

// Update edita nome, WhatsApp e perfil do cliente (RN-03.18). O WhatsApp novo segue a regra do cadastro: não
// pode já ser cliente no mesmo empreendimento. Quem já tinha WhatsApp não fica sem. A troca de nome ou
// WhatsApp guarda os dados de antes no evento de edição.
func (s *BrokerService) Update(ctx context.Context, userID, id string, r lead.UpdateRequest) (lead.Lead, error) {
	valid, err := r.Validate()
	if err != nil {
		return lead.Lead{}, err
	}
	l, err := s.own(ctx, userID, id)
	if err != nil {
		return lead.Lead{}, err
	}
	if valid.Phone == "" && l.Phone != "" {
		return lead.Lead{}, lead.ErrInvalidPhone
	}
	if valid.Phone != "" && valid.Phone != l.Phone {
		other, err := s.repo.FindByPhone(ctx, l.DevelopmentID, valid.Phone)
		switch {
		case err == nil && other.ID != l.ID && other.BrokerID == l.BrokerID:
			return lead.Lead{}, lead.ErrAlreadyYours
		case err == nil && other.ID != l.ID:
			return lead.Lead{}, lead.ErrPhoneTaken
		case err != nil && !errors.Is(err, lead.ErrNotFound):
			return lead.Lead{}, err
		}
	}
	var data []byte
	if valid.Name != l.Name || valid.Phone != l.Phone {
		data, _ = json.Marshal(map[string]map[string]string{"antes": {"nome": l.Name, "whatsapp": l.Phone}})
	}
	return s.repo.Update(ctx, l.ID, valid, data)
}

// Archive é o "excluir" do corretor (RN-03.19): o cliente sai da lista e dos números, mas propostas, vendas e
// histórico ficam guardados. Restore traz de volta.
func (s *BrokerService) Archive(ctx context.Context, userID, id string) (lead.Lead, error) {
	return s.setArchived(ctx, userID, id, true)
}

func (s *BrokerService) Restore(ctx context.Context, userID, id string) (lead.Lead, error) {
	return s.setArchived(ctx, userID, id, false)
}

func (s *BrokerService) setArchived(ctx context.Context, userID, id string, archived bool) (lead.Lead, error) {
	l, err := s.own(ctx, userID, id)
	if err != nil {
		return lead.Lead{}, err
	}
	return s.repo.SetArchived(ctx, l.ID, archived)
}

// Templates devolve os doze modelos (quatro perfis × três tons): o texto do corretor onde ele mudou e o
// padrão no resto (RN-03.20).
func (s *BrokerService) Templates(ctx context.Context, userID string) ([]lead.Template, error) {
	brokerID, err := s.repo.BrokerIDByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	custom, err := s.repo.Templates(ctx, brokerID)
	if err != nil {
		return nil, err
	}
	mine := map[string]lead.Template{}
	for _, t := range custom {
		mine[t.Profile+"/"+t.Tone] = t
	}
	list := make([]lead.Template, 0, len(lead.TemplateProfiles)*len(lead.Tones))
	for _, p := range lead.TemplateProfiles {
		for _, tone := range lead.Tones {
			if t, ok := mine[p+"/"+tone]; ok {
				list = append(list, t)
				continue
			}
			list = append(list, lead.Template{Profile: p, Tone: tone, Body: lead.DefaultTemplate(p, tone)})
		}
	}
	return list, nil
}

func (s *BrokerService) SaveTemplate(ctx context.Context, userID, profile, tone string, r lead.TemplateRequest) (lead.Template, error) {
	if !lead.ValidTemplateKey(profile, tone) {
		return lead.Template{}, lead.ErrTemplateKey
	}
	valid, err := r.Validate()
	if err != nil {
		return lead.Template{}, err
	}
	brokerID, err := s.repo.BrokerIDByUser(ctx, userID)
	if err != nil {
		return lead.Template{}, err
	}
	t := lead.Template{Profile: profile, Tone: tone, Body: valid.Body, Custom: true}
	return t, s.repo.SaveTemplate(ctx, brokerID, t)
}

func (s *BrokerService) ResetTemplate(ctx context.Context, userID, profile, tone string) (lead.Template, error) {
	if !lead.ValidTemplateKey(profile, tone) {
		return lead.Template{}, lead.ErrTemplateKey
	}
	brokerID, err := s.repo.BrokerIDByUser(ctx, userID)
	if err != nil {
		return lead.Template{}, err
	}
	if err := s.repo.DeleteTemplate(ctx, brokerID, profile, tone); err != nil {
		return lead.Template{}, err
	}
	return lead.Template{Profile: profile, Tone: tone, Body: lead.DefaultTemplate(profile, tone)}, nil
}
