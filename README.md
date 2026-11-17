# neo-api

API do **Neo360**: showroom de empreendimentos com CRM leve para corretores. O corretor gera um link exclusivo por empreendimento; o cliente explora a apresentação e envia a proposta; a conversa e o fechamento são pelo WhatsApp. O corretor ajusta a proposta e registra a venda; o admin vê tudo.

O app web está em [andrelbraga/neo-app](https://github.com/andrelbraga/neo-app).

Go 1.25 · Gin · PostgreSQL 16 (pgx) · JWT · Railway

## Rodar local

```bash
make setup
make db-up && make migrate-up   # PostgreSQL em localhost:5433 + schema + Boa Vista pré-carregado
ADMIN_EMAIL=voce@exemplo.com ADMIN_NAME="Seu nome" ADMIN_PASSWORD='uma-senha-longa' make create-admin
make dev                        # http://localhost:8081/api/health
```

## Testes

```bash
make lint test
TEST_DATABASE_URL=postgres://neo360:neo360@localhost:5433/neo360_test?sslmode=disable make test-integration
make cover                      # cobertura de internal/ (meta 80%)
```

## Deploy (Railway)

O serviço publica a cada push na `main`. Postgres próprio no mesmo projeto do Railway, com as tabelas no schema `neo360` (ADR-0018). Variáveis: `APP_ENV=production`, `DATABASE_URL`, `DB_SCHEMA=neo360`, `JWT_SECRET` (longo e aleatório), `ALLOWED_ORIGINS` (URL do app). As migrations rodam ao subir. Para criar o primeiro admin: `railway ssh --service neo-api` e `ADMIN_EMAIL=... ADMIN_NAME=... ADMIN_PASSWORD='...' ./create-admin`.

## Documentação

- Para agentes e devs: [`CLAUDE.md`](CLAUDE.md) e [`.claude/`](.claude/CLAUDE.md)
- Regras de negócio: [`.claude/business-rules/`](.claude/business-rules/README.md)
- Decisões: [`docs/adr/`](docs/adr/README.md) · Pendências: [`docs/pendencias.md`](docs/pendencias.md)
