import { expect, test } from '@playwright/test';
import { FakeApi } from './fake-api';

let api: FakeApi;

test.beforeEach(async ({ page }) => {
	api = new FakeApi();
	await api.install(page);
});

test('una empresa nueva se registra sola y llega al tablero con su portal', async ({ page }) => {
	await page.goto('/');
	await page.getByRole('banner').getByRole('link', { name: 'Crear cuenta gratis' }).click();
	await expect(page).toHaveURL('/signup');

	await page.getByLabel('Nombre de tu empresa').fill('Ferretería López');
	await page.getByLabel('Tu nombre').fill('Marta Admin');
	await page.getByLabel('Email de trabajo').fill('marta@micolmena.dev');
	await page.getByLabel('Contraseña').fill('secreto123');
	await page.getByRole('button', { name: 'Crear cuenta gratis' }).click();

	// Sin aceptar los términos no se crea nada.
	await expect(page.getByText('Debes aceptar para continuar')).toBeVisible();
	expect(api.signups).toHaveLength(0);

	await page.getByRole('checkbox').check();
	await page.getByRole('button', { name: 'Crear cuenta gratis' }).click();

	await expect(page).toHaveURL(/\/dashboard/);
	await expect(
		page.getByRole('heading', { name: /Bienvenido a MiColmena, Ferretería López/ })
	).toBeVisible();
	await expect(page.getByTestId('portal-url').first()).toContainText('/e/ferreteria-lopez');
	expect(api.signups[0]).toMatchObject({
		company: 'Ferretería López',
		email: 'marta@micolmena.dev',
		accept_terms: true
	});
});

test('la calculadora gratis lleva al registro con el tamaño elegido', async ({ page }) => {
	await page.goto('/signup?people=25');
	await expect(page.getByLabel('¿Cuántas personas trabajan en tu empresa?')).toHaveValue('25');
});

test('con cuenta en varias empresas se elige a cuál entrar', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel('Email').fill('multi@correo.dev');
	await page.getByLabel('Contraseña').fill('secreto123');
	await page.getByRole('button', { name: 'Entrar' }).click();

	await page.getByRole('button', { name: /Taller Luna/ }).click();
	await expect(page).toHaveURL(/\/tickets|\/dashboard/);
	const login = api.requests.filter((r) => r.path === '/auth/login');
	expect(login.at(-1)?.body).toMatchObject({ email: 'multi@correo.dev', org: 'taller-luna' });
});

test('el portal de la empresa muestra su nombre y lleva a registrarse en ella', async ({
	page
}) => {
	await page.goto('/e/ferreteria-lopez');
	await expect(page.getByRole('heading', { name: 'Soporte de Ferretería López' })).toBeVisible();
	await page.getByRole('button', { name: 'Abrir un ticket' }).click();
	await expect(page).toHaveURL('/register?org=ferreteria-lopez');
	await expect(page.getByText('Ferretería López').first()).toBeVisible();

	await page.goto('/e/no-existe');
	await expect(page.getByText('Este portal no existe')).toBeVisible();
});

test('Tu plan muestra el uso y pide ampliar con el precio estimado', async ({ page }) => {
	await api.loginAs(page, 'admin');
	await page.goto('/admin/plan');
	await expect(page.getByTestId('agents-usage')).toHaveText('1 de 1');
	await expect(page.getByText('Plan gratis')).toBeVisible();

	await page.getByLabel('Agentes', { exact: true }).fill('3');
	await expect(page.getByTestId('upgrade-estimate')).toContainText('$298');
	await page.getByRole('button', { name: 'Pedir ampliación' }).click();
	await expect(page.getByText('Recibimos tu solicitud')).toBeVisible();
	expect(api.upgrades[0]).toMatchObject({ agents: 3, people: 10 });
});

test('si la sesión se cierra en otro dispositivo, el login explica por qué', async ({ page }) => {
	const reason = 'tu sesión se cerró porque se entró con tu cuenta en otro dispositivo';
	api.revokeWith = reason;
	await api.loginAs(page, 'agent');
	await page.goto('/dashboard');
	await expect(page).toHaveURL(/\/login/);
	await expect(page.getByText(reason)).toBeVisible();
});

test('la dueña de la plataforma ve y administra las empresas', async ({ page }) => {
	await api.loginAs(page, 'owner');
	await page.goto('/platform');
	await expect(page.getByText('Ferretería López')).toBeVisible();
	await page.getByLabel('Agentes permitidos para Ferretería López (vacío = sin límite)').fill('5');
	await page.getByLabel('Agentes permitidos para Ferretería López (vacío = sin límite)').blur();
	await expect
		.poll(
			() => api.requests.find((r) => r.method === 'PATCH' && r.path === '/platform/orgs/2')?.body
		)
		.toMatchObject({ max_agents: 5 });
});

test('los avisos legales se publican como borrador', async ({ page }) => {
	for (const [path, title] of [
		['/privacidad', 'Aviso de privacidad'],
		['/terminos', 'Términos de servicio']
	]) {
		await page.goto(path);
		await expect(page.getByRole('heading', { level: 1 })).toHaveText(title);
		await expect(page.getByText('Borrador', { exact: false }).first()).toBeVisible();
	}
});

test('un agente no ve las secciones de plataforma ni de plan', async ({ page }) => {
	await api.loginAs(page, 'agent');
	await page.goto('/dashboard');
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Tu plan' })).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Empresas' })).toHaveCount(0);
});
