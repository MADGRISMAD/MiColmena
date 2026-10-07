-- Catálogo de servicios: lo que las personas pueden pedir con un formulario (alta de usuario,
-- equipo nuevo, acceso a un sistema…). Cada pedido crea un ticket de tipo «solicitud».
CREATE TABLE service_categories (
    id          BIGSERIAL PRIMARY KEY,
    org_id      BIGINT      NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    icon        TEXT        NOT NULL DEFAULT 'package',
    position    INT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX service_categories_name_key ON service_categories (org_id, lower(name));

CREATE TABLE service_items (
    id              BIGSERIAL PRIMARY KEY,
    org_id          BIGINT      NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    category_id     BIGINT      NOT NULL REFERENCES service_categories (id) ON DELETE CASCADE,
    name            TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    -- Campos del formulario: [{key, label, type, required, options}].
    fields          JSONB       NOT NULL DEFAULT '[]',
    priority        TEXT        NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    -- Categoría del ticket que se crea (opcional).
    ticket_category TEXT        NOT NULL DEFAULT '',
    active          BOOLEAN     NOT NULL DEFAULT true,
    position        INT         NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX service_items_category_idx ON service_items (category_id);

-- Tipo de ticket: incidente (algo falló) o solicitud (se pidió algo del catálogo).
ALTER TABLE tickets ADD COLUMN kind TEXT NOT NULL DEFAULT 'incident' CHECK (kind IN ('incident', 'request'));
ALTER TABLE tickets ADD COLUMN catalog_item_id BIGINT REFERENCES service_items (id) ON DELETE SET NULL;
CREATE INDEX tickets_kind_idx ON tickets (org_id, kind, id DESC);

-- Activos (equipos): inventario con responsable, estado y garantía.
CREATE TABLE assets (
    id             BIGSERIAL PRIMARY KEY,
    org_id         BIGINT        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    tag            TEXT          NOT NULL,
    name           TEXT          NOT NULL,
    category       TEXT          NOT NULL DEFAULT 'computer'
                                 CHECK (category IN ('computer', 'phone', 'monitor', 'network', 'peripheral', 'software', 'other')),
    model          TEXT          NOT NULL DEFAULT '',
    serial         TEXT          NOT NULL DEFAULT '',
    state          TEXT          NOT NULL DEFAULT 'in_stock' CHECK (state IN ('in_stock', 'in_use', 'in_repair', 'retired')),
    assigned_to    BIGINT        REFERENCES users (id) ON DELETE SET NULL,
    location       TEXT          NOT NULL DEFAULT '',
    purchase_date  DATE,
    purchase_cost  NUMERIC(12,2) CHECK (purchase_cost >= 0),
    warranty_until DATE,
    notes          TEXT          NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX assets_tag_key ON assets (org_id, lower(tag));
CREATE INDEX assets_org_state_idx ON assets (org_id, state);
CREATE INDEX assets_assigned_idx ON assets (assigned_to);

CREATE TABLE asset_events (
    id         BIGSERIAL PRIMARY KEY,
    asset_id   BIGINT      NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    actor_id   BIGINT      REFERENCES users (id) ON DELETE SET NULL,
    kind       TEXT        NOT NULL,
    old_value  TEXT        NOT NULL DEFAULT '',
    new_value  TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX asset_events_asset_idx ON asset_events (asset_id, id);

-- Un ticket puede tratar de uno o varios activos, y un activo acumula su historial de tickets.
CREATE TABLE ticket_assets (
    ticket_id BIGINT NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
    asset_id  BIGINT NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    PRIMARY KEY (ticket_id, asset_id)
);
CREATE INDEX ticket_assets_asset_idx ON ticket_assets (asset_id);
