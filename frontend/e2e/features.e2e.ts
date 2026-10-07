import { expect, test } from '@playwright/test';
import { FakeApi, users } from './fake-api';

let api: FakeApi;

test.beforeEach(async ({ page }) => {
	api = new FakeApi();
	await api.install(page);
});

test('acciones masivas sobre varios tickets', async ({ page }) => {
	await api.loginAs(page, 'agent');
	await page.goto('/tickets');

	await page.getByLabel('Seleccionar #1').check();
	await page.getByLabel('Seleccionar #2').check();
	const bar = page.getByRole('toolbar', { name: 'Acciones para los tickets seleccionados' });
	await expect(bar).toContainText('2 seleccionados');
	await bar.getByLabel('Cambiar estado').selectOption('waiting');

	await expect(page.getByText('Estado: En espera: 2 tickets')).toBeVisible();
	const req = api.requests.find((r) => r.path === '/tickets/bulk');
	expect(req?.body).toEqual({ ids: [1, 2], status: 'waiting' });
	await expect(bar).toHaveCount(0);
});

test('respuesta guardada con variables y cambio de estado', async ({ page }) => {
	api.macros.push({
		id: 50,
		title: 'Pedir captura',
		body: 'Hola {{solicitante}}, ¿nos mandas una captura del {{ticket}}?',
		status: 'waiting',
		internal: false,
		created_at: new Date().toISOString(),
		updated_at: new Date().toISOString()
	});
	await api.loginAs(page, 'agent');
	await page.goto('/tickets/1');

	await page.getByRole('button', { name: 'Respuestas guardadas' }).click();
	await page.getByRole('button', { name: /Pedir captura/ }).click();
	await expect(page.getByLabel('Responder')).toHaveValue(
		'Hola Ana, ¿nos mandas una captura del #1?'
	);
	await expect(page.getByText('Al enviar, el estado pasará a')).toBeVisible();

	await page.getByRole('button', { name: 'Enviar respuesta' }).click();
	await expect(page.getByText('cambió el estado de Abierto a En espera')).toBeVisible();
	expect(api.tickets[0].status).toBe('waiting');
});

test('el historial aparece en la conversación', async ({ page }) => {
	await api.loginAs(page, 'agent');
	await page.goto('/tickets/1');
	await page.getByLabel('Estado').selectOption('in_progress');
	await expect(page.getByText('cambió el estado de Abierto a En curso')).toBeVisible();
	await page.getByRole('button', { name: 'Asignarme' }).click();
	await expect(page.getByText(`asignó el ticket a ${users.agent.name}`)).toBeVisible();
});

test('adjuntar un archivo a la respuesta', async ({ page }) => {
	await api.loginAs(page, 'customer');
	await page.goto('/tickets/1');

	await page.locator('input[type=file]').setInputFiles({
		name: 'captura.txt',
		mimeType: 'text/plain',
		buffer: Buffer.from('hola mundo!!')
	});
	await expect(page.getByRole('list', { name: 'Archivos para adjuntar' })).toContainText(
		'captura.txt'
	);
	await page.getByLabel('Responder').fill('Te mando la captura.');
	await page.getByRole('button', { name: 'Enviar respuesta' }).click();

	await expect(page.getByRole('list', { name: 'Archivos adjuntos' })).toContainText('captura.txt');
	expect(api.attachments[0]).toMatchObject({ filename: 'captura.txt', comment_id: 1 });
});

test('el cliente valora un ticket resuelto', async ({ page }) => {
	api.tickets[0].status = 'resolved';
	await api.loginAs(page, 'customer');
	await page.goto('/tickets/1');

	await expect(page.getByText('¿Cómo te atendimos?')).toBeVisible();
	await page.getByRole('button', { name: 'Bien' }).click();
	await page.getByLabel(/Algo que quieras contarnos/).fill('Muy rápidos');
	await page.getByRole('button', { name: 'Enviar valoración' }).click();
	await expect(page.getByText('Gracias por tu valoración', { exact: true })).toBeVisible();
	expect(api.tickets[0]).toMatchObject({
		satisfaction: 'good',
		satisfaction_comment: 'Muy rápidos'
	});
});

test('guardar una vista y verla en la barra lateral', async ({ page }) => {
	await api.loginAs(page, 'agent');
	await page.goto('/tickets?priority=urgent');

	page.once('dialog', (dialog) => dialog.accept('Urgentes'));
	await page.getByRole('button', { name: 'Guardar vista' }).click();
	const link = page.getByRole('link', { name: 'Urgentes', exact: true });
	await expect(link).toBeVisible();
	await expect(link).toHaveAttribute('aria-current', 'page');
	expect(api.views[0]).toMatchObject({ name: 'Urgentes', query: '?priority=urgent' });
});

test('la campana muestra las notificaciones y lleva al ticket', async ({ page }) => {
	api.notifications.push({
		id: 1,
		ticket_id: 2,
		actor: { id: 2, name: 'Ana Cliente' },
		kind: 'comment',
		summary: 'Ana Cliente respondió',
		ticket_title: 'Factura duplicada',
		read_at: null,
		created_at: new Date().toISOString(),
		user_id: users.agent.id
	} as never);
	await api.loginAs(page, 'agent');
	await page.goto('/dashboard');

	const bell = page.getByRole('button', { name: 'Notificaciones (1 sin leer)' });
	await expect(bell).toBeVisible();
	await bell.click();
	await page.getByRole('button', { name: /Ana Cliente respondió/ }).click();
	await expect(page).toHaveURL('/tickets/2');
});

test('el enlace de administración solo lo ve un administrador', async ({ page }) => {
	await api.loginAs(page, 'agent');
	await page.goto('/tickets');
	await expect(page.getByRole('link', { name: 'Respuestas guardadas' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Usuarios' })).toHaveCount(0);
});

test('el administrador ve la gestión de usuarios', async ({ page }) => {
	await api.loginAs(page, 'admin');
	await page.goto('/admin/users');
	await expect(page.getByRole('heading', { name: 'Usuarios' })).toBeVisible();
	await expect(page.getByText('ana@micolmena.dev')).toBeVisible();
});

test('recuperar la contraseña desde el login', async ({ page }) => {
	await page.goto('/login');
	await page.getByRole('link', { name: '¿Olvidaste tu contraseña?' }).click();
	await expect(page.getByRole('heading', { name: 'Recuperar contraseña' })).toBeVisible();
	await page.getByLabel('Email').fill('ana@micolmena.dev');
	await page.getByRole('button', { name: 'Enviar enlace' }).click();
	await expect(page.getByText(/recibirás el enlace/)).toBeVisible();
	expect(api.requests.find((r) => r.path === '/auth/forgot')?.body).toEqual({
		email: 'ana@micolmena.dev'
	});
});

test('la ayuda publicada se lee sin iniciar sesión', async ({ page }) => {
	api.articles.push({
		id: 9,
		title: 'Cómo pagar una factura',
		body: 'Entra en Facturación y elige Pagar.',
		category: 'Facturación',
		published: true,
		author: null,
		org: { slug: 'ferreteria-lopez', name: 'Ferretería López' },
		created_at: new Date().toISOString(),
		updated_at: new Date().toISOString()
	});
	await page.goto('/help');
	await page.getByRole('link', { name: /Cómo pagar una factura/ }).click();
	await expect(page.getByRole('heading', { name: 'Cómo pagar una factura' })).toBeVisible();
	await expect(page.getByText('Entra en Facturación y elige Pagar.')).toBeVisible();
});
