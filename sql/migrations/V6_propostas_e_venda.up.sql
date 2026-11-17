-- ADR-0015: a negociação fecha pelo WhatsApp; a plataforma guarda as propostas (do cliente e os ajustes do
-- corretor) e o desfecho (venda fechada ou sem venda). O chat da página saiu: a tabela messages fica,
-- sem uso, para não perder as conversas já registradas.

ALTER TABLE leads DROP CONSTRAINT leads_stage_check;
ALTER TABLE leads ADD CONSTRAINT leads_stage_check CHECK (stage IN ('novo', 'em_atendimento', 'proposta', 'fechado', 'encerrado'));

CREATE TABLE proposals (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id        UUID NOT NULL REFERENCES leads (id) ON DELETE CASCADE,
    author         TEXT NOT NULL CHECK (author IN ('cliente', 'corretor')),
    based_on       UUID REFERENCES proposals (id) ON DELETE SET NULL,
    condition      JSONB NOT NULL DEFAULT '{}',
    summary        TEXT NOT NULL DEFAULT '',
    note           TEXT NOT NULL DEFAULT '',
    wants_contract BOOLEAN NOT NULL DEFAULT false,
    seen_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX proposals_lead_idx ON proposals (lead_id, created_at);

-- Desfecho: venda fechada (valor, proposta de base, data) ou sem venda (motivo).
ALTER TABLE leads
    ADD COLUMN outcome_at        TIMESTAMPTZ,
    ADD COLUMN sale_value        NUMERIC(14, 2),
    ADD COLUMN sale_proposal_id  UUID REFERENCES proposals (id) ON DELETE SET NULL,
    ADD COLUMN outcome_note      TEXT NOT NULL DEFAULT '';

-- Propostas já enviadas pela apresentação (eventos "proposta") viram registros, já vistos.
INSERT INTO proposals (lead_id, author, condition, summary, note, wants_contract, seen_at, created_at)
SELECT e.lead_id, 'cliente',
       COALESCE(CASE WHEN jsonb_typeof(e.data -> 'condicao') = 'object' THEN e.data -> 'condicao' END, '{}'),
       COALESCE(e.data ->> 'resumo', ''), COALESCE(e.data ->> 'observacao', ''),
       COALESCE((e.data ->> 'autorizou_contrato')::boolean, false), now(), e.created_at
FROM lead_events e WHERE e.kind = 'proposta';
