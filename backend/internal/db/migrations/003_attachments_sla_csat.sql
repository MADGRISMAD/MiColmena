-- Archivos adjuntos. El contenido vive en disco (UPLOAD_DIR); aquí solo los datos.
CREATE TABLE attachments (
    id           BIGSERIAL PRIMARY KEY,
    ticket_id    BIGINT      NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
    -- NULL = adjunto de la descripción del ticket.
    comment_id   BIGINT      REFERENCES ticket_comments (id) ON DELETE CASCADE,
    uploader_id  BIGINT      NOT NULL REFERENCES users (id),
    filename     TEXT        NOT NULL,
    content_type TEXT        NOT NULL,
    size         BIGINT      NOT NULL,
    storage_key  TEXT        NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX attachments_ticket_idx ON attachments (ticket_id, id);

-- Acuerdos de nivel de servicio por prioridad, en minutos de reloj (no solo horario laboral).
CREATE TABLE sla_policies (
    priority               TEXT PRIMARY KEY
                           CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    first_response_minutes INT  NOT NULL CHECK (first_response_minutes > 0),
    resolution_minutes     INT  NOT NULL CHECK (resolution_minutes > 0)
);
INSERT INTO sla_policies (priority, first_response_minutes, resolution_minutes) VALUES
    ('urgent', 60, 240),
    ('high', 240, 1440),
    ('medium', 480, 2880),
    ('low', 1440, 5760);

-- Primera respuesta pública de un agente.
ALTER TABLE tickets ADD COLUMN first_response_at TIMESTAMPTZ;
UPDATE tickets t SET first_response_at = (
    SELECT min(c.created_at) FROM ticket_comments c JOIN users u ON u.id = c.author_id
    WHERE c.ticket_id = t.id AND NOT c.internal AND u.role IN ('agent', 'admin') AND c.author_id <> t.requester_id
);

-- Encuesta de satisfacción del cliente al resolver.
ALTER TABLE tickets ADD COLUMN satisfaction TEXT NOT NULL DEFAULT ''
    CHECK (satisfaction IN ('', 'good', 'bad'));
ALTER TABLE tickets ADD COLUMN satisfaction_comment TEXT NOT NULL DEFAULT '';
ALTER TABLE tickets ADD COLUMN rated_at TIMESTAMPTZ;
