CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT        NOT NULL,
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'customer'
                  CHECK (role IN ('admin', 'agent', 'customer')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- El email se compara sin distinguir mayúsculas.
CREATE UNIQUE INDEX users_email_key ON users (lower(email));
CREATE INDEX users_role_idx ON users (role);

CREATE TABLE tickets (
    id            BIGSERIAL PRIMARY KEY,
    title         TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    status        TEXT        NOT NULL DEFAULT 'open'
                  CHECK (status IN ('open', 'in_progress', 'waiting', 'resolved', 'closed')),
    priority      TEXT        NOT NULL DEFAULT 'medium'
                  CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    category      TEXT        NOT NULL DEFAULT '',
    requester_id  BIGINT      NOT NULL REFERENCES users (id),
    assignee_id   BIGINT      REFERENCES users (id) ON DELETE SET NULL,
    -- Campos personalizados por cliente o categoría, sin cambiar el esquema.
    custom_fields JSONB       NOT NULL DEFAULT '{}'::jsonb,
    search        TSVECTOR GENERATED ALWAYS AS (
                      setweight(to_tsvector('spanish', title), 'A') ||
                      setweight(to_tsvector('spanish', description), 'B')
                  ) STORED,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at   TIMESTAMPTZ
);

-- Índices pensados para las consultas de la lista de tickets (orden por id descendente).
CREATE INDEX tickets_status_idx    ON tickets (status, id DESC);
CREATE INDEX tickets_assignee_idx  ON tickets (assignee_id, status, id DESC);
CREATE INDEX tickets_requester_idx ON tickets (requester_id, id DESC);
CREATE INDEX tickets_search_idx    ON tickets USING GIN (search);

CREATE TABLE ticket_comments (
    id         BIGSERIAL PRIMARY KEY,
    ticket_id  BIGINT      NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
    author_id  BIGINT      NOT NULL REFERENCES users (id),
    body       TEXT        NOT NULL,
    -- Las notas internas solo las ven agentes y administradores.
    internal   BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ticket_comments_ticket_idx ON ticket_comments (ticket_id, id);
