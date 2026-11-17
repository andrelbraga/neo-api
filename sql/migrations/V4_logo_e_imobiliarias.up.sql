-- Logo das empresas: a apresentação mostra o logo da imobiliária do corretor junto ao da incorporadora.
-- Imobiliárias pré-cadastradas (ADR-0010): por ora, só a Omni, a única citada no protótipo (index.html,
-- "Imobiliária Omni", img/imobiliaria-omni.png). Novas imobiliárias entram por migration até a tela voltar.
ALTER TABLE organizations ADD COLUMN logo_url TEXT NOT NULL DEFAULT '';

UPDATE organizations SET logo_url = '/logos/incorporadora-dwell.jpg' WHERE kind = 'incorporadora' AND name = 'Dwell';

INSERT INTO organizations (kind, name, logo_url)
SELECT 'imobiliaria', 'Omni', '/logos/imobiliaria-omni.png'
WHERE NOT EXISTS (SELECT 1 FROM organizations WHERE kind = 'imobiliaria' AND name = 'Omni');
