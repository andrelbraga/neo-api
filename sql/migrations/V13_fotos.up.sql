-- ADR-0029: fotos no bucket S3 (Railway Bucket neo-files). O banco guarda só a chave do arquivo.
-- Foto do corretor, que aparece no cartão "Seu consultor" da apresentação. Vazio = sem foto (as iniciais).
ALTER TABLE brokers ADD COLUMN photo_key TEXT NOT NULL DEFAULT '';

-- Fotos dos veículos que o cliente anexa à proposta. Chegam antes da proposta (lead_id) e ficam ligadas a ela
-- quando o cliente envia (proposal_id); só as ligadas aparecem para o corretor.
CREATE TABLE lead_photos (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id      UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    proposal_id  UUID REFERENCES proposals(id) ON DELETE SET NULL,
    object_key   TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp')),
    size_bytes   INTEGER NOT NULL CHECK (size_bytes > 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX lead_photos_lead_idx ON lead_photos (lead_id);
CREATE INDEX lead_photos_proposal_idx ON lead_photos (proposal_id);
