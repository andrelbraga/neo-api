-- Cidade das empresas e imobiliárias pré-cadastradas da Serra Gaúcha e de Porto Alegre (ADR-0010, P-12).
-- Nomes levantados em 16/11/2026 em sites próprios, portais e notícias (fontes em docs/pendencias.md, P-12).
-- Estar na lista não significa parceria: confirme CRECI-J e o interesse antes de ligar um corretor.
-- Sem logo: a apresentação mostra o nome até o logo autorizado ser incluído em /logos/.
ALTER TABLE organizations ADD COLUMN city TEXT NOT NULL DEFAULT '';

UPDATE organizations SET city = 'Gramado' WHERE kind = 'incorporadora' AND name = 'Dwell';

INSERT INTO organizations (kind, name, city)
SELECT 'imobiliaria', v.name, v.city
FROM (VALUES
    ('Imobiliária Gramadense', 'Gramado'),
    ('Serra Prime Imobiliária', 'Gramado'),
    ('Imobiliária Ghisleni', 'Gramado'),
    ('TM Consultoria Imobiliária', 'Canela'),
    ('B&B Locação e Venda de Imóveis', 'Canela'),
    ('Serra & Lar', 'Canela'),
    ('Vieitez & Souza', 'Canela'),
    ('Lar na Serra Imobiliária', 'Canela'),
    ('Becker House Imobiliária', 'Canela'),
    ('Realiza Imóveis', 'Canela'),
    ('Imobiliária Bento Alves', 'Caxias do Sul'),
    ('Nobille Negócios Imobiliários', 'Caxias do Sul'),
    ('Kiko Imóveis', 'Bento Gonçalves'),
    ('Top House Transações Imobiliárias', 'Bento Gonçalves'),
    ('Pontual Imóveis', 'Bento Gonçalves'),
    ('Imobiliária FAF', 'Bento Gonçalves'),
    ('Auxiliadora Predial', 'Porto Alegre'),
    ('Foxter Cia. Imobiliária', 'Porto Alegre'),
    ('Guarida Imóveis', 'Porto Alegre'),
    ('Da Cás Lançamentos', 'Porto Alegre')
) AS v (name, city)
WHERE NOT EXISTS (SELECT 1 FROM organizations o WHERE o.kind = 'imobiliaria' AND o.name = v.name);
