-- Multiempresa: cada empresa (organización) tiene sus usuarios, tickets y configuración,
-- y nadie ve los datos de otra. La empresa 1 es la de la plataforma (MiColmena): ahí quedan
-- los datos que ya existían, sus administradores reciben las solicitudes de demo y los
-- administradores permanentes de esa empresa gestionan todas las demás.
CREATE TABLE organizations (
    id                BIGSERIAL PRIMARY KEY,
    name              TEXT        NOT NULL,
    -- Dirección del portal: micolmena.com/e/<slug>
    slug              TEXT        NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    -- Personas en la empresa, declaradas al registrarse (definen el precio).
    people            INT         NOT NULL DEFAULT 10 CHECK (people > 0),
    -- Agentes (agentes + administradores) permitidos por el plan; NULL = sin límite.
    max_agents        INT         CHECK (max_agents > 0),
    suspended         BOOLEAN     NOT NULL DEFAULT false,
    terms_accepted_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO organizations (id, name, slug, people, max_agents) VALUES (1, 'MiColmena', 'micolmena', 10, NULL);
SELECT setval('organizations_id_seq', 1);

-- Usuarios: el mismo email puede existir en varias empresas (por ejemplo, cliente de dos).
ALTER TABLE users ADD COLUMN org_id BIGINT NOT NULL DEFAULT 1 REFERENCES organizations (id) ON DELETE CASCADE;
ALTER TABLE users ALTER COLUMN org_id DROP DEFAULT;
DROP INDEX users_email_key;
CREATE UNIQUE INDEX users_email_key ON users (org_id, lower(email));
CREATE INDEX users_org_role_idx ON users (org_id, role);
CREATE INDEX users_email_idx ON users (lower(email));

ALTER TABLE tickets ADD COLUMN org_id BIGINT NOT NULL DEFAULT 1 REFERENCES organizations (id) ON DELETE CASCADE;
ALTER TABLE tickets ALTER COLUMN org_id DROP DEFAULT;
CREATE INDEX tickets_org_idx ON tickets (org_id, id DESC);
CREATE INDEX tickets_org_status_idx ON tickets (org_id, status, id DESC);

ALTER TABLE categories ADD COLUMN org_id BIGINT NOT NULL DEFAULT 1 REFERENCES organizations (id) ON DELETE CASCADE;
ALTER TABLE categories ALTER COLUMN org_id DROP DEFAULT;
DROP INDEX categories_name_key;
CREATE UNIQUE INDEX categories_name_key ON categories (org_id, lower(name));

ALTER TABLE macros ADD COLUMN org_id BIGINT NOT NULL DEFAULT 1 REFERENCES organizations (id) ON DELETE CASCADE;
ALTER TABLE macros ALTER COLUMN org_id DROP DEFAULT;
CREATE INDEX macros_org_idx ON macros (org_id);

ALTER TABLE articles ADD COLUMN org_id BIGINT NOT NULL DEFAULT 1 REFERENCES organizations (id) ON DELETE CASCADE;
ALTER TABLE articles ALTER COLUMN org_id DROP DEFAULT;
CREATE INDEX articles_org_idx ON articles (org_id, updated_at DESC);

-- Cada empresa tiene sus propios plazos de SLA.
ALTER TABLE sla_policies ADD COLUMN org_id BIGINT NOT NULL DEFAULT 1 REFERENCES organizations (id) ON DELETE CASCADE;
ALTER TABLE sla_policies ALTER COLUMN org_id DROP DEFAULT;
ALTER TABLE sla_policies DROP CONSTRAINT sla_policies_pkey;
ALTER TABLE sla_policies ADD PRIMARY KEY (org_id, priority);

-- Las solicitudes de demo pueden venir de una empresa que ya usa MiColmena (pedir ampliar el plan).
ALTER TABLE leads ADD COLUMN org_id BIGINT REFERENCES organizations (id) ON DELETE SET NULL;
