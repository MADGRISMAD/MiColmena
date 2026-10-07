import { describe, expect, it } from 'vitest';
import { estimate, money, STARTING_PRICE } from './pricing';

describe('estimate', () => {
	it('suma la cuota de la empresa y los agentes por tramos', () => {
		const e = estimate(8, 100);
		expect(e.companyFee).toBe(699);
		expect(e.agentLines).toEqual([
			{ count: 5, price: 149 },
			{ count: 3, price: 129 }
		]);
		expect(e.monthly).toBe(699 + 5 * 149 + 3 * 129);
		expect(e.custom).toBe(false);
	});

	it('la misma cantidad de agentes cuesta más en una empresa más grande', () => {
		expect(estimate(2, 800).monthly).toBeGreaterThan(estimate(2, 20).monthly);
	});

	it('el pago anual cobra 10 de 12 meses', () => {
		const e = estimate(3, 25);
		expect(e.annualTotal).toBe(e.monthly * 10);
		expect(e.monthlyAnnual).toBe(Math.round((e.monthly * 10) / 12));
	});

	it('pide cotización fuera de la calculadora', () => {
		expect(estimate(51, 100).custom).toBe(true);
		expect(estimate(10, 1001).custom).toBe(true);
	});

	it('1 agente y hasta 10 personas es gratis', () => {
		for (const people of [5, 10]) {
			const e = estimate(1, people);
			expect(e.free).toBe(true);
			expect(e.monthly).toBe(0);
			expect(e.annualTotal).toBe(0);
		}
		expect(estimate(1, 11).free).toBe(false);
	});

	it('crecer desde el plan gratis no es un salto: el primer agente sigue incluido', () => {
		const e = estimate(2, 10);
		expect(e.agentLines).toEqual([
			{ count: 1, price: 0 },
			{ count: 1, price: 149 }
		]);
		expect(e.monthly).toBe(149);
		expect(e.free).toBe(false);
		expect(STARTING_PRICE).toBe(149);
		expect(estimate(6, 10).monthly).toBe(4 * 149 + 129);
		expect(money(1299)).toBe('$1,299');
	});
});
