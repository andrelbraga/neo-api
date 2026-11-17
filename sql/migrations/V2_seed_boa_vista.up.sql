-- Empreendimento pré-carregado: Boa Vista Residencial, com os textos do protótipo (prototipo/boa-vista-104a).
-- Condições comerciais e preços ficam fora até a regra ser confirmada (docs/pendencias.md, P-03).
-- Os arquivos de imagem e o tour 3D são servidos pelo app em /showroom/boa-vista/.

INSERT INTO organizations (kind, name)
VALUES ('incorporadora', 'Dwell');

INSERT INTO developments (slug, name, developer_id, city, state, status, published, headline, description,
                          cover_image_url, tour_url, gallery, highlights)
SELECT 'boa-vista-residencial',
       'Boa Vista Residencial',
       o.id,
       'Gramado',
       'RS',
       'lancamento',
       TRUE,
       'Invista em Gramado',
       'Um apartamento na planta com potencial de valorização, locação e uso próprio na Serra Gaúcha.',
       '/showroom/boa-vista/img/hero-gramado.jpg',
       '/showroom/boa-vista/maquete.html?embed=1',
       '[
         {"url": "/showroom/boa-vista/img/fachada.jpg", "caption": "Fachada"},
         {"url": "/showroom/boa-vista/img/aerea.jpg", "caption": "Vista aérea"},
         {"url": "/showroom/boa-vista/img/sala.jpg", "caption": "Sala do decorado"},
         {"url": "/showroom/boa-vista/img/salao-gourmet-1.jpg", "caption": "Salão gourmet"},
         {"url": "/showroom/boa-vista/img/salao-gourmet-2.jpg", "caption": "Salão gourmet"},
         {"url": "/showroom/boa-vista/img/planta-54m2-referencia.webp", "caption": "Planta de referência, 54 m²"}
       ]'::jsonb,
       '[
         {"label": "Área privativa", "value": "54 m²"},
         {"label": "Dormitórios", "value": "2"},
         {"label": "Vaga", "value": "1"},
         {"label": "Torres", "value": "4: Aurora, Belvedere, Crepúsculo e Domus"},
         {"label": "Vagas no total", "value": "156, uma por apartamento, 61 cobertas"},
         {"label": "Área comum", "value": "Salão gourmet decorado e mobiliado"}
       ]'::jsonb
FROM organizations o
WHERE o.kind = 'incorporadora' AND o.name = 'Dwell';
