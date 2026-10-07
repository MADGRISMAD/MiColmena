import { expect, test } from '@playwright/test';
import { FakeApi } from './fake-api';

let api: FakeApi;

test.beforeEach(async ({ page }) => {
	api = new FakeApi();
	await api.install(page);
});

test('los precios están en pesos y el pago anual aplica el descuento', async ({ page }) => {
	await page.goto('/');
	const pricing = page.locator('#precios');
	await pricing.scrollIntoViewIfNeeded();

	await expect(pricing.getByRole('button', { name: /Anual/ })).toHaveAttribute(
		'aria-pressed',
		'true'
	);
	await expect(pricing.getByText('$208', { exact: true })).toBeVisible();
	await pricing.getByRole('button', { name: 'Mensual' }).click();
	await expect(pricing.getByText('$249', { exact: true })).toBeVisible();
	await expect(pricing.getByText('A la medida', { exact: true })).toBeVisible();
});

test('elegir un plan abre el formulario de demo con ese plan y se envía', async ({ page }) => {
	await page.goto('/');
	await page.locator('#precios').getByRole('button', { name: 'Solicitar demo' }).nth(1).click();

	const form = page.locator('#demo');
	await expect(form.getByLabel('Plan que te interesa')).toHaveValue('profesional');

	await form.getByRole('button', { name: 'Solicitar mi demo' }).click();
	await expect(form.getByText('Escribe tu nombre')).toBeVisible();
	expect(api.leads).toHaveLength(0);

	await form.getByLabel('Nombre').fill('Rosa Méndez');
	await form.getByLabel('Empresa').fill('Ferretería López');
	await form.getByLabel('Email de trabajo').fill('rosa@ferreteria.mx');
	await form.getByLabel('¿Cuántas personas atenderán tickets?').selectOption('4-10');
	await form.getByRole('button', { name: 'Solicitar mi demo' }).click();

	await expect(form.getByText('¡Gracias, Rosa!')).toBeVisible();
	expect(api.leads[0]).toMatchObject({
		name: 'Rosa Méndez',
		company: 'Ferretería López',
		email: 'rosa@ferreteria.mx',
		team_size: '4-10',
		plan: 'profesional',
		website: ''
	});
});

test('las preguntas frecuentes se abren y cierran', async ({ page }) => {
	await page.goto('/');
	const question = page.getByRole('button', {
		name: '¿Cobran por los clientes que abren tickets?'
	});
	await question.click();
	await expect(question).toHaveAttribute('aria-expanded', 'true');
	await expect(page.getByText('Solo pagas por los agentes')).toBeVisible();
	await question.click();
	await expect(page.getByText('Solo pagas por los agentes')).toBeHidden();
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
