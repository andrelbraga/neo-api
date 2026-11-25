-- RN-03.17 a RN-03.19 (ADR-0025): perfil de compra do cliente, arquivamento (o "excluir" do corretor, sem apagar
-- nada) e os modelos de mensagem do WhatsApp que o corretor ajusta, por perfil e tom.
ALTER TABLE leads ADD COLUMN profile TEXT NOT NULL DEFAULT '' CHECK (profile IN ('', 'moradia', 'investimento', 'veraneio'));
ALTER TABLE leads ADD COLUMN archived_at TIMESTAMPTZ;

-- Só os textos que o corretor mudou; os demais vêm do padrão da API.
CREATE TABLE message_templates (
    broker_id  UUID NOT NULL REFERENCES brokers (id) ON DELETE CASCADE,
    profile    TEXT NOT NULL CHECK (profile IN ('geral', 'moradia', 'investimento', 'veraneio')),
    tone       TEXT NOT NULL CHECK (tone IN ('formal', 'proximo', 'direto')),
    body       TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (broker_id, profile, tone)
);
