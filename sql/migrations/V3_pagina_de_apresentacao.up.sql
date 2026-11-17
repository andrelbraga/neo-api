-- Página de apresentação própria do empreendimento (HTML servido pelo app). Quando existe, o showroom
-- mostra essa página no lugar do modelo padrão. O Boa Vista usa a apresentação do protótipo (ADR-0008).
ALTER TABLE developments ADD COLUMN presentation_url TEXT NOT NULL DEFAULT '';

UPDATE developments SET presentation_url = '/showroom/boa-vista/index.html' WHERE slug = 'boa-vista-residencial';
