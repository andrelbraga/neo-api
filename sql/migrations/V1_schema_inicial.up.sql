-- Schema inicial do Neo360: contas, empresas, corretores, empreendimentos e o CRM leve (clientes, eventos e chat).

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('admin', 'corretor')),
    active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

-- Incorporadoras e imobiliárias têm o mesmo cadastro; o tipo separa as telas do admin.
CREATE TABLE organizations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind       TEXT NOT NULL CHECK (kind IN ('incorporadora', 'imobiliaria')),
    name       TEXT NOT NULL,
    document   TEXT NOT NULL DEFAULT '',
    creci      TEXT NOT NULL DEFAULT '',
    phone      TEXT NOT NULL DEFAULT '',
    email      TEXT NOT NULL DEFAULT '',
    website    TEXT NOT NULL DEFAULT '',
    active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX organizations_kind_idx ON organizations (kind, name);

-- O corretor é a pessoa por trás da conta de papel "corretor". CRECI é opcional (RN-01.2);
-- o código entra no link exclusivo e não muda (RN-02.1).
CREATE TABLE brokers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    first_name TEXT NOT NULL,
    last_name  TEXT NOT NULL,
    phone      TEXT NOT NULL,
    creci      TEXT NOT NULL DEFAULT '',
    code       TEXT NOT NULL UNIQUE,
    agency_id  UUID REFERENCES organizations (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE developments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    developer_id    UUID REFERENCES organizations (id) ON DELETE SET NULL,
    city            TEXT NOT NULL DEFAULT '',
    state           TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'lancamento' CHECK (status IN ('lancamento', 'em_obras', 'pronto')),
    published       BOOLEAN NOT NULL DEFAULT FALSE,
    headline        TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    cover_image_url TEXT NOT NULL DEFAULT '',
    tour_url        TEXT NOT NULL DEFAULT '',
    gallery         JSONB NOT NULL DEFAULT '[]',
    highlights      JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Cliente que abriu o link de um corretor. O dono é o corretor do link da primeira visita e não muda
-- (RN-03.1). O navegador guarda só o token; o banco guarda o hash dele.
CREATE TABLE leads (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    development_id   UUID NOT NULL REFERENCES developments (id) ON DELETE CASCADE,
    broker_id        UUID NOT NULL REFERENCES brokers (id) ON DELETE CASCADE,
    token_hash       TEXT NOT NULL UNIQUE,
    name             TEXT NOT NULL DEFAULT '',
    phone            TEXT NOT NULL DEFAULT '',
    stage            TEXT NOT NULL DEFAULT 'novo' CHECK (stage IN ('novo', 'em_atendimento', 'proposta', 'encerrado')),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX leads_broker_idx ON leads (broker_id, last_activity_at DESC);
CREATE INDEX leads_development_idx ON leads (development_id, last_activity_at DESC);

CREATE TABLE lead_events (
    id         BIGSERIAL PRIMARY KEY,
    lead_id    UUID NOT NULL REFERENCES leads (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,
    data       JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX lead_events_lead_idx ON lead_events (lead_id, created_at);

CREATE TABLE messages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id    UUID NOT NULL REFERENCES leads (id) ON DELETE CASCADE,
    author     TEXT NOT NULL CHECK (author IN ('cliente', 'corretor')),
    body       TEXT NOT NULL,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX messages_lead_idx ON messages (lead_id, created_at);
