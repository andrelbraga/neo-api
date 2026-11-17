# Neo360 — instruções do projeto

> Após cada alteração, registre o que mudou em `.claude/specs.md`.

## O que é

Showroom de empreendimentos (lançamentos, decorados) com CRM leve para corretores. O corretor se cadastra, gera um **link exclusivo** por empreendimento e envia ao cliente pelo WhatsApp. O cliente abre o showroom (fotos, destaques, maquete 3D), tira dúvidas e envia a proposta; a conversa e o fechamento são pelo WhatsApp do corretor (ADR-0015). O corretor vê quem reagiu e o que fez, ajusta a proposta e registra a venda ou o encerramento. O **admin** vê tudo e cadastra empreendimentos e profissionais (incorporadoras e imobiliárias fora da tela por ora, ADR-0009).

Primeiro empreendimento pré-carregado: **Boa Vista Residencial** (Gramado/RS), migrado do protótipo em `prototipo/boa-vista-104a/`.

## Arquivos de referência

| Arquivo | Quando ler |
|---|---|
| `.claude/rules.md` | **Sempre** — regras técnicas |
| `.claude/design.md` | Ao criar domínio, rota, tabela ou tela |
| `.claude/style.md` | Ao escrever código (Go, SQL, TS) |
| `.claude/specs.md` | Para saber o que mudou e o que está em andamento |
| `.claude/business-rules/` | Antes de mexer em conta, link, showroom, cliente, proposta ou venda |
| `docs/pendencias.md` | Antes de implementar regra de negócio sem fonte |
| `docs/adr/` | Decisões de arquitetura e produto (MADR) |

## Estrutura

Este repositório é a **API** (`neo-api`). O app web e a apresentação do Boa Vista estão em [`neo-app`](https://github.com/andrelbraga/neo-app). As decisões (`docs/adr/`), as pendências e as regras de negócio (`.claude/business-rules/`) existem nos dois repositórios: ao mudar uma, atualize a cópia no outro (ADR-0017).

```
cmd/        api, migrate e create-admin
internal/   config, domain (account, broker, organization, development, lead), infra, pkg
sql/        migrations (V{n}_*.up/down.sql)
docs/       ADRs e pendências (cópia compartilhada)
```

Go 1.25 · Gin · PostgreSQL 16 (pgx) · JWT · Railway.

Domínios (`internal/domain/`): `account` (login, cadastro aberto, perfil), `broker` (corretores), `organization` (incorporadoras e imobiliárias), `development` (empreendimentos), `lead` (showroom público, eventos, propostas, desfecho, CRM do corretor e visão do admin).

## Rodar local

```bash
make db-up && make migrate-up             # Postgres em localhost:5433
ADMIN_EMAIL=... ADMIN_NAME=... ADMIN_PASSWORD='...' make create-admin
make dev                                  # API em http://localhost:8081
```

O app (`neo-app`) roda em http://localhost:5174 e chama esta API.

Portas diferentes do neonflow (5432/8080/5173) para os dois rodarem juntos.

## Estado em 16/11/2026

Implementado e testado (API com 95,7% de cobertura em `internal/` com integração; app com 33 testes):
cadastro do corretor sem CRECI obrigatório; link exclusivo por empreendimento; showroom com destaques, galeria, tour 3D e contato pelo WhatsApp; rastreio do que o cliente abre; dono fixo pelo primeiro link; proposta com versões (cliente e ajustes do corretor) e desfecho (venda fechada ou sem venda); CRM do corretor (quadro, ficha com Resumo, Propostas e Atividades, WhatsApp); admin com cadastro de incorporadoras, imobiliárias, empreendimentos e profissionais e visão de todos os clientes.

Desde 16/11/2026 (ADR-0008) o Boa Vista abre com a apresentação do protótipo, ligada à API por `public/showroom/neo360-bridge.js` (repositório neo-app): ao mudar o HTML da apresentação, confira a ponte.

Fora desta versão (ver `docs/pendencias.md`): valores comerciais editáveis no admin, aviso com o painel fechado, upload de imagens, recuperação de senha, campanhas Neo360.
