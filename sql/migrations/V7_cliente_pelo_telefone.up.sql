-- ADR-0016: o mesmo cliente (mesmo WhatsApp no mesmo empreendimento) é um cliente só, mesmo vindo de
-- navegadores diferentes. O cadastro mais antigo fica; os outros apontam para ele (merged_into) e o token
-- deles continua funcionando. A proposta guarda o nome e o WhatsApp com que foi enviada.

ALTER TABLE leads ADD COLUMN merged_into UUID REFERENCES leads (id) ON DELETE CASCADE;
CREATE INDEX leads_phone_idx ON leads (development_id, phone) WHERE phone <> '' AND merged_into IS NULL;

ALTER TABLE proposals
    ADD COLUMN client_name  TEXT NOT NULL DEFAULT '',
    ADD COLUMN client_phone TEXT NOT NULL DEFAULT '';

UPDATE proposals p SET client_name = l.name, client_phone = l.phone
FROM leads l WHERE l.id = p.lead_id AND p.author = 'cliente';
