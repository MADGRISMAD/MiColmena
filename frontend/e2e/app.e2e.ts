import { expect, test } from '@playwright/test';
import { FakeApi, users } from './fake-api';

let api: FakeApi;

test.beforeEach(async ({ page }) => {
	api = new FakeApi();
	await api.install(page);
});

test('la portada lleva a iniciar sesión', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await page.getByRole('banner').getByRole('link', { name: 'Iniciar sesión' }).click();
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

test('el login muestra el aviso cuando el servidor limita los intentos', async ({ page }) => {
	const message = 'demasiados intentos, inténtalo de nuevo en 15 minutos';
	await page.route('**/api/auth/login', (route) =>
		route.fulfill({ status: 429, headers: { 'Retry-After': '900' }, json: { error: message } })
	);

	await page.goto('/login');
	await page.getByLabel('Email').fill(users.agent.email);
	await page.getByLabel('Contraseña').fill('secreto123');
	await page.getByRole('button', { name: 'Entrar' }).click();

	await expect(page.getByText(message)).toBeVisible();
	await expect(page).toHaveURL('/login');
	// Se puede volver a intentarlo: el botón no queda bloqueado.
	await expect(page.getByRole('button', { name: 'Entrar' })).toBeEnabled();
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
	await expect(page.getByRole('heading', { name: 'La app se cierra sola' })).toBeVisible();
	await expect(page.getByRole('navigation', { name: 'Ruta' })).toContainText('#4');
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
	// Los clientes no pueden crear notas internas: no ven la pestaña.
	await expect(page.getByRole('tab', { name: 'Nota interna' })).toHaveCount(0);
});

test('el agente cambia el estado, asigna y deja una nota interna', async ({ page }) => {
	await api.loginAs(page, 'agent');
	await page.goto('/tickets/1');

	await page.getByLabel('Estado').selectOption('in_progress');
	await expect(page.getByText('Estado: En curso')).toBeVisible();

	await page.getByLabel('Asignado a').selectOption(String(users.agent.id));
	await expect(page.getByText('Ticket asignado')).toBeVisible();
	expect(api.tickets[0]).toMatchObject({ status: 'in_progress', assignee: { id: users.agent.id } });

	await page.getByRole('tab', { name: 'Nota interna' }).click();
	await expect(page.getByRole('tab', { name: 'Nota interna' })).toHaveAttribute(
		'aria-selected',
		'true'
	);
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

test('el agente resuelve, reabre y se asigna un ticket con un clic', async ({ page }) => {
	await api.loginAs(page, 'agent');
	await page.goto('/tickets/2');

	await page.getByRole('button', { name: 'Resolver' }).click();
	await expect(page.getByText('Ticket resuelto')).toBeVisible();
	expect(api.tickets[1].status).toBe('resolved');

	await page.getByRole('button', { name: 'Reabrir' }).click();
	await expect(page.getByText('Ticket reabierto')).toBeVisible();
	expect(api.tickets[1].status).toBe('open');

	await page.getByRole('button', { name: 'Asignarme' }).click();
	await expect(page.getByLabel('Asignado a')).toHaveValue(String(users.agent.id));
	await expect(page.getByRole('button', { name: 'Asignarme' })).toHaveCount(0);
});

test('Ctrl+Enter envía la respuesta y tras una nota interna se vuelve a respuesta pública', async ({
	page
}) => {
	await api.loginAs(page, 'agent');
	await page.goto('/tickets/1');

	await page.getByRole('tab', { name: 'Nota interna' }).click();
	await page.getByLabel('Responder').fill('Solo para el equipo.');
	await page.getByLabel('Responder').press('Control+Enter');

	await expect(page.getByText('Solo para el equipo.')).toBeVisible();
	expect(api.comments.at(-1)).toMatchObject({ body: 'Solo para el equipo.', internal: true });
	// Para no mandar por error otra nota interna, el editor vuelve a "Respuesta pública".
	await expect(page.getByRole('tab', { name: 'Respuesta pública' })).toHaveAttribute(
		'aria-selected',
		'true'
	);
});

test('las pestañas de estado y las vistas filtran la lista', async ({ page }) => {
	await api.loginAs(page, 'agent');
	await page.goto('/tickets');

	await page.getByRole('button', { name: 'En curso' }).click();
	await expect(page).toHaveURL('/tickets?status=in_progress');
	await expect(page.getByRole('button', { name: 'En curso' })).toHaveAttribute(
		'aria-pressed',
		'true'
	);

	await page.getByRole('link', { name: 'Sin asignar' }).click();
	await expect(page).toHaveURL('/tickets?assignee=none');
	await expect(page.getByRole('heading', { name: 'Sin asignar' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Sin asignar' })).toHaveAttribute(
		'aria-current',
		'page'
	);
});

test('la tecla / lleva a la búsqueda desde cualquier pantalla', async ({ page }) => {
	await api.loginAs(page, 'agent');
	await page.goto('/dashboard');
	await expect(page.getByText('Tickets sin resolver')).toBeVisible();

	await page.keyboard.press('/');
	await expect(page.getByLabel('Buscar tickets')).toBeFocused();
	await page.keyboard.type('impresora');
	await page.keyboard.press('Enter');
	await expect(page).toHaveURL('/tickets?q=impresora');
	await expect(page.getByRole('link', { name: 'Error en la impresora' })).toBeVisible();
});

test.describe('en el móvil', () => {
	test.use({ viewport: { width: 390, height: 844 } });

	test('la lista muestra tarjetas que caben en la pantalla', async ({ page }) => {
		await api.loginAs(page, 'agent');
		await page.goto('/tickets');

		const card = page.getByRole('listitem').filter({ hasText: 'No puedo iniciar sesión' });
		await expect(card).toBeVisible();
		const box = await card.boundingBox();
		expect(box && box.x + box.width).toBeLessThanOrEqual(390);
		// El estado se ve completo dentro de la tarjeta.
		const status = card.getByText('Abierto');
		await expect(status).toBeInViewport({ ratio: 1 });
	});

	test('el menú lateral se abre y se cierra al navegar', async ({ page }) => {
		await api.loginAs(page, 'agent');
		await page.goto('/tickets');

		await page.getByRole('button', { name: 'Abrir menú' }).click();
		const menu = page.getByRole('dialog', { name: 'Menú' });
		await expect(menu).toBeVisible();
		await menu.getByRole('link', { name: 'Dashboard' }).click();
		await expect(page).toHaveURL('/dashboard');
		await expect(menu).toBeHidden();
	});
});
