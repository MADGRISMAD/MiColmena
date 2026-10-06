-- Notificaciones dentro de la aplicación (la campana).
CREATE TABLE notifications (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    ticket_id  BIGINT      REFERENCES tickets (id) ON DELETE CASCADE,
    actor_id   BIGINT      REFERENCES users (id) ON DELETE SET NULL,
    -- new_ticket, assigned, comment, mention, status
    kind       TEXT        NOT NULL,
    summary    TEXT        NOT NULL,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user_idx ON notifications (user_id, id DESC);
CREATE INDEX notifications_unread_idx ON notifications (user_id) WHERE read_at IS NULL;

-- Cola de correos: la API los guarda en la misma transacción que el cambio y un
-- proceso aparte los envía con reintentos. Si el servidor de correo falla, no se pierde nada.
CREATE TABLE email_outbox (
    id         BIGSERIAL PRIMARY KEY,
    to_email   TEXT        NOT NULL,
    subject    TEXT        NOT NULL,
    body       TEXT        NOT NULL,
    attempts   INT         NOT NULL DEFAULT 0,
    last_error TEXT        NOT NULL DEFAULT '',
    send_after TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX email_outbox_pending_idx ON email_outbox (send_after) WHERE sent_at IS NULL;

-- Cada usuario decide si quiere recibir correos además de la campana.
ALTER TABLE users ADD COLUMN email_notifications BOOLEAN NOT NULL DEFAULT true;

-- Enlaces para restablecer la contraseña. Solo se guarda el hash del token.
CREATE TABLE password_resets (
    token_hash TEXT PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX password_resets_user_idx ON password_resets (user_id);
