import { describe, expect, it } from 'vitest';
import { planPrice, plans } from './pricing';

describe('planPrice', () => {
	const pro = plans.find((p) => p.id === 'profesional')!;

	it('muestra el precio en pesos', () => {
		expect(planPrice(pro, false)).toMatch(/^\$249$/);
	});

	it('el pago anual cobra 10 de 12 meses', () => {
		expect(planPrice(pro, true)).toMatch(/^\$208$/);
	});

	it('el plan a la medida no tiene precio', () => {
		expect(
			planPrice(
				plans.find((p) => p.id === 'empresa')!,
				false
			)
		).toBeNull();
	});
});
