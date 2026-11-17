DELETE FROM organizations o
WHERE o.kind = 'imobiliaria' AND o.logo_url = '' AND o.city IN ('Gramado', 'Canela', 'Caxias do Sul', 'Bento Gonçalves', 'Porto Alegre')
  AND NOT EXISTS (SELECT 1 FROM brokers b WHERE b.agency_id = o.id);
ALTER TABLE organizations DROP COLUMN city;
