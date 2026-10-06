-- Cuentas desactivadas: no pueden entrar ni usar tokens ya emitidos.
ALTER TABLE users ADD COLUMN active BOOLEAN NOT NULL DEFAULT true;
-- Los tokens emitidos antes de este momento dejan de valer (cambio de contraseña o desactivación).
ALTER TABLE users ADD COLUMN tokens_valid_after TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01T00:00:00Z';

-- Historial de cambios de cada ticket.
CREATE TABLE ticket_events (
    id         BIGSERIAL PRIMARY KEY,
    ticket_id  BIGINT      NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
    actor_id   BIGINT      REFERENCES users (id) ON DELETE SET NULL,
    -- created, status, priority, assignee, category, title, tags
    kind       TEXT        NOT NULL,
    old_value  TEXT        NOT NULL DEFAULT '',
    new_value  TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ticket_events_ticket_idx ON ticket_events (ticket_id, id);

-- Los tickets que ya existían quedan con su evento de creación.
INSERT INTO ticket_events (ticket_id, actor_id, kind, created_at)
SELECT id, requester_id, 'created', created_at FROM tickets;

-- Etiquetas libres para agrupar y filtrar tickets.
ALTER TABLE tickets ADD COLUMN tags TEXT[] NOT NULL DEFAULT '{}';
CREATE INDEX tickets_tags_idx ON tickets USING GIN (tags);

-- Categorías que administra un administrador. El ticket guarda el nombre como texto.
CREATE TABLE categories (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX categories_name_key ON categories (lower(name));

-- Las categorías que ya se usaban pasan a la lista.
INSERT INTO categories (name)
SELECT DISTINCT ON (lower(category)) category FROM tickets WHERE category <> '';

-- Respuestas guardadas (macros) para los agentes.
CREATE TABLE macros (
    id         BIGSERIAL PRIMARY KEY,
    title      TEXT        NOT NULL,
    body       TEXT        NOT NULL,
    -- Estado al que pasa el ticket al enviar la respuesta; vacío = no cambia.
    status     TEXT        NOT NULL DEFAULT ''
               CHECK (status IN ('', 'open', 'in_progress', 'waiting', 'resolved', 'closed')),
    internal   BOOLEAN     NOT NULL DEFAULT false,
    created_by BIGINT      REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
