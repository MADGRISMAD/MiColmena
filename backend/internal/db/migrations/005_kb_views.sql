-- Base de conocimiento: artículos de ayuda. Los publicados se ven sin iniciar sesión.
CREATE TABLE articles (
    id         BIGSERIAL PRIMARY KEY,
    title      TEXT        NOT NULL,
    body       TEXT        NOT NULL DEFAULT '',
    category   TEXT        NOT NULL DEFAULT '',
    published  BOOLEAN     NOT NULL DEFAULT false,
    author_id  BIGINT      REFERENCES users (id) ON DELETE SET NULL,
    search     TSVECTOR GENERATED ALWAYS AS (
                   setweight(to_tsvector('spanish', title), 'A') ||
                   setweight(to_tsvector('spanish', body), 'B')
               ) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX articles_search_idx ON articles USING GIN (search);

-- Vistas guardadas: combinaciones de filtros que cada usuario fija en su barra lateral.
CREATE TABLE saved_views (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    query      TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX saved_views_user_idx ON saved_views (user_id, id);
