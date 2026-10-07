-- La calculadora de precios envía cuántos agentes y cuántas personas tiene la empresa.
-- team_size y plan quedan para las solicitudes anteriores.
ALTER TABLE leads ADD COLUMN agents INT NOT NULL DEFAULT 0 CHECK (agents >= 0);
ALTER TABLE leads ADD COLUMN people INT NOT NULL DEFAULT 0 CHECK (people >= 0);
