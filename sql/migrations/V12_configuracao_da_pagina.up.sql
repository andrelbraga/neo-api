-- ADR-0027: o admin edita a página de apresentação do empreendimento (mini CMS). A configuração guarda as
-- seções ocultas, os blocos desligados (como o 3D) e os textos trocados; a página aplica ao abrir.
ALTER TABLE developments ADD COLUMN page_config JSONB NOT NULL DEFAULT '{}';
