ALTER TABLE leads DROP COLUMN outcome_note, DROP COLUMN sale_proposal_id, DROP COLUMN sale_value, DROP COLUMN outcome_at;
DROP TABLE proposals;
UPDATE leads SET stage = 'encerrado' WHERE stage = 'fechado';
ALTER TABLE leads DROP CONSTRAINT leads_stage_check;
ALTER TABLE leads ADD CONSTRAINT leads_stage_check CHECK (stage IN ('novo', 'em_atendimento', 'proposta', 'encerrado'));
