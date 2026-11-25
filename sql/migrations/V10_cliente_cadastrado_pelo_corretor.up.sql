-- RN-03.16: o corretor cadastra clientes (um a um ou por planilha CSV) e manda a cada um um link de convite.
-- origin separa quem chegou pelo link de quem o corretor cadastrou; invite_code é o código do link de convite
-- do cliente cadastrado (na primeira abertura, a visita cai na ficha dele).
ALTER TABLE leads ADD COLUMN origin TEXT NOT NULL DEFAULT 'link' CHECK (origin IN ('link', 'corretor'));
ALTER TABLE leads ADD COLUMN invite_code TEXT UNIQUE;
