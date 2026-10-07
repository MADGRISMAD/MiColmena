import { expect, test } from '@playwright/test';
import { FakeApi } from './fake-api';

let api: FakeApi;

test.beforeEach(async ({ page }) => {
	api = new FakeApi();
	await api.install(page);
});

test('el cliente explora y busca en el catálogo', async ({ page }) => {
	await api.loginAs(page, 'customer');
	await page.goto('/catalog');

	await expect(
		page.getByRole('heading', { name: 'Catálogo de servicios', level: 1 })
	).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Accesos y seguridad' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Equipo y dispositivos' })).toBeVisible();
	await expect(page.getByRole('link', { name: /Restablecer contraseña/ })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Editar catálogo' })).toHaveCount(0);

	await page.getByLabel('Buscar un servicio').fill('laptop');
	await expect(page.getByRole('link', { name: /Equipo de cómputo nuevo/ })).toBeVisible();
	await expect(page.getByRole('link', { name: /Restablecer contraseña/ })).toHaveCount(0);
	await expect(page.getByRole('heading', { name: 'Accesos y seguridad' })).toHaveCount(0);

	await page.getByLabel('Buscar un servicio').fill('zzzz');
	await expect(page.getByText('Ningún servicio coincide')).toBeVisible();
});

test('pedir un servicio valida los campos obligatorios y crea una solicitud', async ({ page }) => {
	await api.loginAs(page, 'customer');
	await page.goto('/catalog');
	await page.getByRole('link', { name: /Acceso a un sistema/ }).click();
	await expect(page.getByRole('heading', { name: 'Acceso a un sistema', level: 1 })).toBeVisible();

	await page.getByRole('button', { name: 'Enviar solicitud' }).click();
	await expect(page.getByText('Este campo es obligatorio')).toHaveCount(2);
	expect(api.tickets.at(-1)?.kind).not.toBe('request');

	await page.getByLabel('¿A qué sistema o carpeta?').fill('Carpeta Finanzas');
	await page.getByLabel('Nivel de acceso').selectOption('Solo lectura');
	await page.getByLabel('Notas adicionales (opcional)').fill('Es urgente para el cierre');
	await page.getByRole('button', { name: 'Enviar solicitud' }).click();

	const ticket = api.tickets.at(-1)!;
	await expect(page).toHaveURL(new RegExp(`/tickets/${ticket.id}$`));
	expect(ticket.kind).toBe('request');
	expect(ticket.catalog_item).toEqual({ id: 1, name: 'Acceso a un sistema' });
	expect(ticket.description).toContain('Carpeta Finanzas');
	expect(ticket.description).toContain('Solo lectura');
	expect(ticket.description).toContain('Es urgente para el cierre');
	await expect(page.getByText('Acceso a un sistema').first()).toBeVisible();

	// En la lista aparece marcada como solicitud.
	await page.goto('/tickets');
	await expect(page.getByRole('table').getByText('Solicitud', { exact: true })).toBeVisible();
});

test('un servicio inexistente muestra un aviso', async ({ page }) => {
	await api.loginAs(page, 'customer');
	await page.goto('/catalog/999');
	await expect(page.getByText('Este servicio no existe')).toBeVisible();
});

test('el admin crea, edita, desactiva y elimina servicios y categorías', async ({ page }) => {
	page.on('dialog', (d) => d.accept());
	await api.loginAs(page, 'admin');
	await page.goto('/catalog');
	await page.getByRole('link', { name: 'Editar catálogo' }).click();
	await expect(page).toHaveURL(/\/admin\/catalog$/);

	// Nueva categoría.
	await page.getByRole('button', { name: 'Nueva categoría' }).first().click();
	await page.getByLabel('Nombre de la categoría').fill('Software');
	await page.getByLabel('Icono').selectOption('wrench');
	await page.getByRole('button', { name: 'Guardar categoría' }).click();
	await expect(page.getByRole('heading', { name: 'Software' })).toBeVisible();
	expect(api.catalog.at(-1)?.name).toBe('Software');

	// Nuevo servicio con una lista y un texto obligatorio.
	await page.getByRole('button', { name: 'Nuevo servicio en Software' }).click();
	await page.getByLabel('Nombre del servicio').fill('Instalar programa');
	await page.getByRole('button', { name: 'Añadir campo' }).click();
	const first = page.getByRole('group', { name: 'Campo 1' });
	await first.getByLabel('Etiqueta').fill('Programa');
	await first.getByLabel('Tipo').selectOption('select');
	await first.getByLabel('Opciones (una por línea)').fill('Office\nPhotoshop');
	await page.getByRole('button', { name: 'Añadir campo' }).click();
	const second = page.getByRole('group', { name: 'Campo 2' });
	await second.getByLabel('Etiqueta').fill('Justificación');
	await second.getByLabel('Obligatorio').check();
	await page.getByRole('button', { name: 'Guardar servicio' }).click();

	await expect(page.getByText('Instalar programa')).toBeVisible();
	const created = api.catalog.at(-1)!.items[0];
	expect(created.fields.map((f) => [f.label, f.type, f.required])).toEqual([
		['Programa', 'select', false],
		['Justificación', 'text', true]
	]);
	expect(created.fields[0].options).toEqual(['Office', 'Photoshop']);

	// Editar: renombrar y subir el segundo campo.
	await page.getByRole('button', { name: 'Editar servicio Instalar programa' }).click();
	await page.getByLabel('Nombre del servicio').fill('Instalar software');
	await page.getByRole('button', { name: 'Subir campo 2' }).click();
	await page.getByRole('button', { name: 'Guardar servicio' }).click();
	await expect(page.getByText('Instalar software')).toBeVisible();
	expect(api.catalog.at(-1)!.items[0].fields[0].label).toBe('Justificación');
	expect(api.catalog.at(-1)!.items[0].fields[0].key).toBeTruthy();

	// Desactivar: el cliente ya no lo ve.
	await page.getByRole('button', { name: 'Editar servicio Instalar software' }).click();
	await page.getByLabel('Activo').uncheck();
	await page.getByRole('button', { name: 'Guardar servicio' }).click();
	await expect(page.getByText('Inactivo')).toBeVisible();

	await api.loginAs(page, 'customer');
	await page.goto('/catalog');
	await expect(page.getByRole('link', { name: /Restablecer contraseña/ })).toBeVisible();
	await expect(page.getByText('Instalar software')).toHaveCount(0);

	// Eliminar servicio y categoría.
	await api.loginAs(page, 'admin');
	await page.goto('/admin/catalog');
	await page.getByRole('button', { name: 'Eliminar servicio Instalar software' }).click();
	await expect(page.getByText('Instalar software')).toHaveCount(0);
	await page.getByRole('button', { name: 'Eliminar categoría Software' }).click();
	await expect(page.getByRole('heading', { name: 'Software' })).toHaveCount(0);
	expect(api.catalog.map((c) => c.name)).not.toContain('Software');
});

test('quien no es admin no puede abrir el editor del catálogo', async ({ page }) => {
	await api.loginAs(page, 'customer');
	await page.goto('/admin/catalog');
	await expect(page).toHaveURL(/\/tickets$/);
	await expect(page.getByRole('button', { name: 'Nueva categoría' })).toHaveCount(0);
});
