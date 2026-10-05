import { expect, test } from '@playwright/test';
import { FakeApi, users } from './fake-api';

let api: FakeApi;

test.beforeEach(async ({ page }) => {
	api = new FakeApi();
	await api.install(page);
});

test('la portada invita a registrarse o iniciar sesión', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await page.getByRole('link', { name: 'Iniciar sesión' }).click();
	await expect(page).toHaveURL('/login');
});

test('sin sesión, una página privada lleva al login y vuelve después', async ({ page }) => {
	await page.goto('/tickets/new');
	await expect(page).toHaveURL('/login?redirect=%2Ftickets%2Fnew');

	await page.getByLabel('Email').fill(users.agent.email);
	await page.getByLabel('Contraseña').fill('secreto123');
	await page.getByRole('button', { name: 'Entrar' }).click();

	await expect(page).toHaveURL('/tickets/new');
	await expect(page.getByRole('heading', { name: 'Nuevo ticket' })).toBeVisible();
});

test('el login valida en el navegador y muestra el error del servidor', async ({ page }) => {
	await page.goto('/login');

	await page.getByRole('button', { name: 'Entrar' }).click();
	await expect(page.getByText('Introduce un email válido')).toBeVisible();
	await expect(page.getByText('Introduce tu contraseña')).toBeVisible();
	expect(api.requests).toHaveLength(0);

	await page.getByLabel('Email').fill(users.agent.email);
	await page.getByLabel('Contraseña').fill('incorrecta');
	await page.getByRole('button', { name: 'Entrar' }).click();
	await expect(page.getByText('email o contraseña incorrectos')).toBeVisible();
});

test('el agente entra al dashboard con las estadísticas', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel('Email').fill(users.agent.email);
	await page.getByLabel('Contraseña').fill('secreto123');
	await page.getByRole('button', { name: 'Entrar' }).click();

	await expect(page).toHaveURL('/dashboard');
	await expect(page.getByText('Tickets sin resolver')).toBeVisible();
	await expect(page.getByText('5,5 h')).toBeVisible();
});

test('el registro avisa si las contraseñas no coinciden', async ({ page }) => {
	await page.goto('/register');
	await page.getByLabel('Nombre').fill('Ana');
	await page.getByLabel('Email').fill('ana@x.com');
	await page.getByLabel('Contraseña', { exact: true }).fill('secreto123');
	await page.getByLabel('Repite la contraseña').fill('otra-cosa');
	await page.getByRole('button', { name: 'Crear cuenta' }).click();
	await expect(page.getByText('Las contraseñas no coinciden')).toBeVisible();
});

test('el cliente solo ve sus tickets y puede filtrarlos', async ({ page }) => {
	await api.loginAs(page, 'customer');
	await page.goto('/tickets');

	await expect(page.getByRole('heading', { name: 'Mis tickets' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'No puedo iniciar sesión' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Error en la impresora' })).toHaveCount(0);

	await page.getByLabel('Buscar tickets').fill('factura');
	await page.getByLabel('Buscar tickets').press('Enter');
	await expect(page).toHaveURL('/tickets?q=factura');
	await expect(page.getByRole('link', { name: 'Factura duplicada' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'No puedo iniciar sesión' })).toHaveCount(0);
});

test('crear un ticket y responder en la conversación', async ({ page }) => {
	await api.loginAs(page, 'customer');
	await page.goto('/tickets/new');

	await page.getByRole('button', { name: 'Crear ticket' }).click();
	await expect(page.getByText('El título es obligatorio')).toBeVisible();

	await page.getByLabel('Título').fill('La app se cierra sola');
	await page.getByLabel('Descripción').fill('Al abrir el menú.');
	await page.getByLabel('Prioridad').selectOption('high');
	await page.getByRole('button', { name: 'Crear ticket' }).click();

	await expect(page).toHaveURL(/\/tickets\/4$/);
	await expect(page.getByRole('heading', { name: '#4 · La app se cierra sola' })).toBeVisible();
	expect(api.requests.find((r) => r.method === 'POST' && r.path === '/tickets')?.body).toEqual({
		title: 'La app se cierra sola',
		description: 'Al abrir el menú.',
		priority: 'high',
		category: ''
	});

	await page.getByLabel('Responder').fill('Adjunto más detalles.');
	await page.getByRole('button', { name: 'Enviar respuesta' }).click();
	await expect(page.getByText('Adjunto más detalles.')).toBeVisible();
	await expect(page.getByLabel('Responder')).toHaveValue('');
	// Los clientes no pueden crear notas internas.
	await expect(page.getByText('Nota interna (el cliente no la ve)')).toHaveCount(0);
});

test('el agente cambia el estado, asigna y deja una nota interna', async ({ page }) => {
	await api.loginAs(page, 'agent');
	await page.goto('/tickets/1');

	await page.getByLabel('Estado').selectOption('in_progress');
	await expect(page.getByText('Estado: En curso')).toBeVisible();

	await page.getByLabel('Asignado a').selectOption(String(users.agent.id));
	await expect(page.getByText('Ticket asignado')).toBeVisible();
	expect(api.tickets[0]).toMatchObject({ status: 'in_progress', assignee: { id: users.agent.id } });

	await page.getByLabel('Nota interna (el cliente no la ve)').check();
	await page.getByLabel('Responder').fill('Revisar logs del servidor.');
	await page.getByRole('button', { name: 'Añadir nota' }).click();
	await expect(page.getByText('Revisar logs del servidor.')).toBeVisible();
	expect(api.comments[0]).toMatchObject({ internal: true });
});

test('un cliente no puede ver tickets ajenos', async ({ page }) => {
	await api.loginAs(page, 'customer');
	await page.goto('/tickets/3');
	await expect(page.getByText('Este ticket no existe o no tienes acceso a él.')).toBeVisible();
});
