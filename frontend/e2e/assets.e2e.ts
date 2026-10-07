import { expect, test } from '@playwright/test';
import { FakeApi } from './fake-api';

let api: FakeApi;

test.beforeEach(async ({ page }) => {
	api = new FakeApi();
	await api.install(page);
});

test('el resumen muestra tarjetas, gráficas y la lista, y se filtra por tarjeta, estado y búsqueda', async ({
	page
}) => {
	await api.loginAs(page, 'agent');
	await page.goto('/assets');

	await expect(page.getByRole('heading', { name: 'Activos' })).toBeVisible();
	const tiles = page.getByRole('list', { name: 'Resumen del inventario' });
	await expect(tiles.getByRole('listitem').filter({ hasText: 'Total de equipos' })).toContainText(
		'3'
	);
	await expect(tiles.getByRole('listitem').filter({ hasText: 'En uso' })).toContainText('1');
	await expect(tiles.getByRole('listitem').filter({ hasText: 'Garantía vencida' })).toContainText(
		'1'
	);
	// Las tarjetas informativas no son enlaces.
	await expect(tiles.getByRole('link', { name: /Con tickets abiertos/ })).toHaveCount(0);
	await expect(tiles.getByRole('link', { name: /Sin responsable/ })).toHaveCount(0);

	// Gráficas con leyenda de texto y valor total.
	await expect(
		page.getByRole('img', { name: /Equipos por categoría: Computadora 1/ })
	).toBeVisible();
	await expect(page.getByRole('list', { name: 'Leyenda de categorías' })).toContainText('Monitor');
	await expect(page.getByRole('list', { name: 'Equipos por estado' })).toContainText(
		'En reparación'
	);
	await expect(page.getByTestId('inventory-value')).toContainText('35,500');

	const rows = page.getByRole('row');
	await expect(rows).toHaveCount(4); // encabezado + 3
	await expect(page.getByRole('link', { name: 'LAP-001' })).toBeVisible();
	await expect(page.getByText('Vencida', { exact: true })).toBeVisible();

	// Tarjeta -> filtro en la URL.
	await tiles.getByRole('link', { name: /En uso/ }).click();
	await expect(page).toHaveURL(/state=in_use/);
	await expect(rows).toHaveCount(2);
	await expect(page.getByRole('link', { name: 'LAP-001' })).toBeVisible();

	await tiles.getByRole('link', { name: /Garantía vencida/ }).click();
	await expect(page).toHaveURL(/warranty=expired/);
	await expect(page.getByRole('link', { name: 'PH-001' })).toBeVisible();
	await expect(rows).toHaveCount(2);

	await page.getByRole('link', { name: 'Quitar filtros' }).click();
	await expect(rows).toHaveCount(4);

	// Selector de estado.
	await page.getByLabel('Filtrar por estado').selectOption('in_repair');
	await expect(page).toHaveURL(/state=in_repair/);
	await expect(page.getByRole('link', { name: 'MON-001' })).toBeVisible();
	await expect(rows).toHaveCount(2);
	await page.getByLabel('Filtrar por estado').selectOption('');

	// Búsqueda con retardo.
	await page.getByLabel('Buscar equipos').fill('iphone');
	await expect(page).toHaveURL(/q=iphone/);
	await expect(page.getByRole('link', { name: 'PH-001' })).toBeVisible();
	await expect(rows).toHaveCount(2);

	await page.getByLabel('Buscar equipos').fill('zzz');
	await expect(page.getByText('Ningún equipo coincide con los filtros')).toBeVisible();
});

test('un cliente no puede abrir los activos', async ({ page }) => {
	await api.loginAs(page, 'customer');
	await page.goto('/assets');
	await expect(page).toHaveURL(/\/tickets/);
	await expect(page.getByRole('link', { name: 'Activos' })).toHaveCount(0);
});

test('sin equipos se invita a agregar el primero', async ({ page }) => {
	api.assets = [];
	await api.loginAs(page, 'agent');
	await page.goto('/assets');
	await expect(page.getByText('Todavía no hay equipos')).toBeVisible();
	await page.getByRole('link', { name: 'Agregar equipo' }).click();
	await expect(page).toHaveURL(/\/assets\/new/);
});

test('crear un equipo valida la etiqueta y lleva a su ficha; al editarlo y asignarlo queda en uso', async ({
	page
}) => {
	await api.loginAs(page, 'agent');
	await page.goto('/assets');
	await page.getByRole('link', { name: 'Nuevo equipo' }).click();
	await expect(page).toHaveURL(/\/assets\/new/);

	// Etiqueta en blanco.
	await page.getByLabel('Nombre', { exact: true }).fill('MacBook Air');
	await page.getByRole('button', { name: 'Crear equipo' }).click();
	await expect(page.getByText('La etiqueta es obligatoria')).toBeVisible();

	// Etiqueta duplicada: error de la API en su campo.
	await page.getByLabel('Etiqueta').fill('lap-001');
	await page.getByRole('button', { name: 'Crear equipo' }).click();
	await expect(page.getByText('ya existe un equipo con esa etiqueta')).toBeVisible();

	await page.getByLabel('Etiqueta').fill('LAP-002');
	await page.getByLabel('Modelo').fill('MacBook Air M3');
	await page.getByLabel('Ubicación').fill('Dirección');
	await page.getByLabel('Costo (MXN)').fill('28000');
	await page.getByLabel('Garantía hasta').fill('2099-05-01');
	await page.getByRole('button', { name: 'Crear equipo' }).click();

	await expect(page).toHaveURL(/\/assets\/\d+$/);
	await expect(page.getByRole('heading', { name: /LAP-002 MacBook Air/ })).toBeVisible();
	expect(
		api.requests.filter((r) => r.path === '/assets' && r.method === 'POST').at(-1)?.body
	).toMatchObject({
		tag: 'LAP-002',
		name: 'MacBook Air',
		model: 'MacBook Air M3',
		purchase_cost: 28000,
		warranty_until: '2099-05-01'
	});
	await expect(page.getByText('En almacén').first()).toBeVisible();

	// Editar: asignarlo a una persona lo pone en uso.
	await page.getByLabel('Responsable').selectOption({ label: 'Ana Cliente' });
	await page.getByRole('button', { name: 'Guardar cambios' }).click();
	await expect(page.getByText('Equipo actualizado')).toBeVisible();
	await expect(page.getByLabel('Estado', { exact: true })).toHaveValue('in_use');
	const patch = api.requests.find((r) => r.method === 'PATCH' && r.path.startsWith('/assets/'));
	expect(patch?.body).toMatchObject({ tag: 'LAP-002', name: 'MacBook Air', assigned_to: 2 });

	await page.getByRole('tab', { name: 'Actividad' }).click();
	const panel = page.getByRole('tabpanel');
	await expect(panel.getByText('Se dio de alta el equipo')).toBeVisible();
	await expect(panel.getByText('Asignado a Ana Cliente').first()).toBeVisible();
	await expect(panel.getByText('Estado: En almacén → En uso')).toBeVisible();
});

test('la pestaña Tickets lista los tickets vinculados al equipo', async ({ page }) => {
	api.ticketAssets.push({ ticket_id: 1, asset_id: 1 });
	await api.loginAs(page, 'agent');
	await page.goto('/assets/1');

	await expect(page.getByRole('heading', { name: /LAP-001 Dell Latitude/ })).toBeVisible();
	await page.getByRole('tab', { name: /Tickets/ }).click();
	const panel = page.getByRole('tabpanel');
	await expect(panel.getByRole('link', { name: 'No puedo iniciar sesión' })).toBeVisible();
	await expect(panel.getByText('Abierto')).toBeVisible();
	await expect(panel.getByRole('link', { name: 'Ver en la lista de tickets' })).toHaveAttribute(
		'href',
		/asset=1/
	);

	// Solo el administrador puede eliminar.
	await page.getByRole('tab', { name: 'General' }).click();
	await expect(page.getByRole('button', { name: 'Eliminar equipo' })).toHaveCount(0);
});

test('un administrador elimina un equipo tras confirmar', async ({ page }) => {
	await api.loginAs(page, 'admin');
	await page.goto('/assets/3');
	page.once('dialog', (d) => d.accept());
	await page.getByRole('button', { name: 'Eliminar equipo' }).click();
	await expect(page).toHaveURL(/\/assets$/);
	await expect(page.getByRole('link', { name: 'MON-001' })).toHaveCount(0);
});

test('soporte vincula y desvincula equipos desde el ticket', async ({ page }) => {
	await api.loginAs(page, 'agent');
	await page.goto('/tickets/1');

	const card = page.getByTestId('ticket-assets');
	await expect(card.getByText('Ningún equipo vinculado.')).toBeVisible();

	// Sugerencia: el equipo asignado a quien abrió el ticket.
	await card.getByRole('button', { name: /LAP-001/ }).click();
	await expect(page.getByText('Equipo LAP-001 vinculado')).toBeVisible();
	await expect(card.getByRole('link', { name: /LAP-001/ })).toBeVisible();
	expect(api.ticketAssets).toContainEqual({ ticket_id: 1, asset_id: 1 });

	// Búsqueda.
	await card.getByLabel('Vincular un equipo').fill('monitor');
	await card.getByRole('button', { name: /MON-001/ }).click();
	await expect(card.getByRole('link', { name: /MON-001/ })).toBeVisible();
	expect(api.ticketAssets).toHaveLength(2);

	await card.getByRole('button', { name: 'Desvincular MON-001' }).click();
	await expect(page.getByText('Equipo MON-001 desvinculado')).toBeVisible();
	await expect(card.getByRole('link', { name: /MON-001/ })).toHaveCount(0);
	expect(api.ticketAssets).toEqual([{ ticket_id: 1, asset_id: 1 }]);
});

test('un cliente no ve la sección de equipos en el ticket', async ({ page }) => {
	await api.loginAs(page, 'customer');
	await page.goto('/tickets/1');
	await expect(page.getByRole('heading', { name: 'No puedo iniciar sesión' })).toBeVisible();
	await expect(page.getByTestId('ticket-assets')).toHaveCount(0);
});

test('Mis equipos en el perfil y ticket sobre un equipo con ?asset=', async ({ page }) => {
	await api.loginAs(page, 'customer');
	await page.goto('/profile');

	const card = page.getByText('Mis equipos', { exact: true });
	await expect(card).toBeVisible();
	await expect(page.getByText('Dell Latitude 5350')).toBeVisible();
	await page.getByRole('link', { name: 'Reportar un problema con LAP-001' }).click();

	await expect(page).toHaveURL(/\/tickets\/new\?asset=1/);
	await expect(page.getByLabel('¿Con qué equipo tienes el problema?')).toHaveValue('1');

	await page.getByLabel('Título').fill('Se apaga sola');
	await page.getByRole('button', { name: 'Crear ticket' }).click();
	await expect(page).toHaveURL(/\/tickets\/\d+$/);
	const create = api.requests.find((r) => r.path === '/tickets' && r.method === 'POST');
	expect(create?.body).toMatchObject({ title: 'Se apaga sola', asset_id: 1 });
});

test('sin equipos asignados no aparece Mis equipos ni el selector', async ({ page }) => {
	api.assets = [];
	await api.loginAs(page, 'customer');
	await page.goto('/profile');
	await expect(page.getByRole('heading', { name: 'Mi perfil' })).toBeVisible();
	await expect(page.getByText('Mis equipos', { exact: true })).toHaveCount(0);

	await page.goto('/tickets/new');
	await expect(page.getByLabel('Título')).toBeVisible();
	await expect(page.getByLabel('¿Con qué equipo tienes el problema?')).toHaveCount(0);
});
