import { expect, test } from '@playwright/test';
import { FakeApi } from './fake-api';

let api: FakeApi;

test.beforeEach(async ({ page }) => {
	api = new FakeApi();
	await api.install(page);
});

test('la calculadora cambia el precio con los agentes y el tamaño de la empresa', async ({
	page
}) => {
	await page.goto('/');
	const pricing = page.locator('#precios');
	const total = pricing.getByTestId('total');
	await pricing.scrollIntoViewIfNeeded();

	// Por defecto: 5 agentes, hasta 100 personas, pago anual (10 de 12 meses).
	await expect(total).toHaveText('$1,203');
	await pricing.getByRole('button', { name: 'Mensual' }).click();
	await expect(total).toHaveText('$1,444');

	// 8 agentes: 5 a $149 y 3 a $129, más la cuota de la empresa.
	await pricing.getByLabel(/^Agentes/).fill('8');
	await expect(total).toHaveText('$1,831');
	await expect(pricing.getByText('3 agentes × $129')).toBeVisible();

	// La misma gente en una empresa más grande cuesta más.
	await pricing.getByLabel(/^Personas en tu empresa/).fill('7');
	await expect(pricing.getByText('Hasta 1,000', { exact: true })).toBeVisible();
	await expect(total).toHaveText('$4,131');

	// Fuera de la calculadora se cotiza.
	await pricing.getByLabel(/^Personas en tu empresa/).fill('8');
	await expect(pricing.getByText('A la medida')).toBeVisible();
	await expect(pricing.getByRole('button', { name: 'Pedir cotización' })).toBeVisible();
});

test('1 agente y hasta 10 personas es gratis para siempre', async ({ page }) => {
	await page.goto('/');
	const pricing = page.locator('#precios');
	await pricing.getByLabel(/^Personas en tu empresa/).fill('1'); // hasta 10
	await pricing.getByLabel(/^Agentes/).fill('1');
	await expect(pricing.getByTestId('total')).toHaveText('Gratis');
	await expect(pricing.getByText('para siempre')).toBeVisible();

	// El segundo agente se cobra; el primero sigue incluido.
	await pricing.getByLabel(/^Agentes/).fill('2');
	await pricing.getByRole('button', { name: 'Mensual' }).click();
	await expect(pricing.getByTestId('total')).toHaveText('$149');
	await expect(pricing.getByText('1 agente incluido')).toBeVisible();

	await pricing.getByLabel(/^Agentes/).fill('1');
	await pricing.getByRole('button', { name: 'Quiero el plan gratis' }).click();
	await expect(page.locator('#demo').getByTestId('quote')).toContainText('plan gratis');
});

test('los agentes nunca superan a las personas de la empresa', async ({ page }) => {
	await page.goto('/');
	const pricing = page.locator('#precios');
	await pricing.getByLabel(/^Personas en tu empresa/).fill('0'); // hasta 5 personas
	await expect(pricing.getByLabel(/^Agentes/)).toHaveValue('5');
	await pricing.getByLabel(/^Agentes/).fill('20');
	await expect(pricing.getByText('Hasta 25', { exact: true })).toBeVisible();
});

test('la demo se pide con el precio calculado', async ({ page }) => {
	await page.goto('/');
	const pricing = page.locator('#precios');
	await pricing.getByLabel(/^Agentes/).fill('8');
	await pricing.getByRole('button', { name: 'Solicitar demo con este precio' }).click();

	const form = page.locator('#demo');
	await expect(form.getByTestId('quote')).toContainText('8 agentes');
	await expect(form.getByTestId('quote')).toContainText('hasta 100 personas');

	await form.getByRole('button', { name: 'Solicitar mi demo' }).click();
	await expect(form.getByText('Escribe tu nombre')).toBeVisible();
	expect(api.leads).toHaveLength(0);

	await form.getByLabel('Nombre').fill('Rosa Méndez');
	await form.getByLabel('Empresa').fill('Ferretería López');
	await form.getByLabel('Email de trabajo').fill('rosa@ferreteria.mx');
	await form.getByRole('button', { name: 'Solicitar mi demo' }).click();

	await expect(form.getByText('¡Gracias, Rosa!')).toBeVisible();
	expect(api.leads[0]).toMatchObject({
		name: 'Rosa Méndez',
		company: 'Ferretería López',
		email: 'rosa@ferreteria.mx',
		agents: 8,
		people: 100,
		website: ''
	});
});

test('las preguntas frecuentes se abren y cierran', async ({ page }) => {
	await page.goto('/');
	const question = page.getByRole('button', { name: /¿Cobran por ticket/ });
	const answer = page.getByText('Puedes recibir los tickets que quieras');
	await question.click();
	await expect(question).toHaveAttribute('aria-expanded', 'true');
	await expect(answer).toBeVisible();
	await question.click();
	await expect(answer).toBeHidden();
});

test.describe('en el móvil', () => {
	test.use({ viewport: { width: 390, height: 844 } });

	test('la portada no se desborda a lo ancho', async ({ page }) => {
		await page.goto('/');
		// El hero recorta lo que sobra, así que se mide el texto en sí, no el ancho de la página.
		for (const el of [
			page.getByRole('heading', { level: 1 }),
			page.getByText('Tickets, SLA, centro de ayuda y reportes')
		]) {
			const box = await el.boundingBox();
			expect(box && box.x + box.width).toBeLessThanOrEqual(390);
		}
		await page.locator('#demo').scrollIntoViewIfNeeded();
		const width = await page.evaluate(() => document.documentElement.scrollWidth);
		expect(width).toBeLessThanOrEqual(390);
	});
});
