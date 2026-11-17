DELETE FROM organizations WHERE kind = 'imobiliaria' AND name = 'Omni'
  AND NOT EXISTS (SELECT 1 FROM brokers b WHERE b.agency_id = organizations.id);
ALTER TABLE organizations DROP COLUMN logo_url;
