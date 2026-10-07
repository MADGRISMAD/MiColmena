-- La plataforma pasa a llamarse BeHIve: la empresa 1 (la de la plataforma) cambia de nombre
-- y de dirección de portal. Las demás empresas no cambian.
UPDATE organizations SET name = 'BeHIve', slug = 'behive' WHERE id = 1 AND slug = 'micolmena';
