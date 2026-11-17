-- ADR-0019: a apresentação tem dois layouts (A e B). O admin escolhe o padrão de cada empreendimento, o
-- corretor pode mandar o link num ou noutro, e cada cliente guarda o layout da primeira visita para comparar
-- qual converte mais.
ALTER TABLE developments ADD COLUMN default_layout TEXT NOT NULL DEFAULT 'a' CHECK (default_layout IN ('a', 'b'));
ALTER TABLE leads ADD COLUMN layout TEXT NOT NULL DEFAULT 'a' CHECK (layout IN ('a', 'b'));
