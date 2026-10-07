-- Solicitudes de demo que llegan desde la landing.
CREATE TABLE leads (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT        NOT NULL,
    company    TEXT        NOT NULL,
    email      TEXT        NOT NULL,
    phone      TEXT        NOT NULL DEFAULT '',
    team_size  TEXT        NOT NULL DEFAULT '',
    plan       TEXT        NOT NULL DEFAULT '',
    message    TEXT        NOT NULL DEFAULT '',
    handled    BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX leads_created_idx ON leads (id DESC);
