import { describe, expect, it } from 'vitest';
import { safeRedirect } from './navigation';

describe('safeRedirect', () => {
	it('acepta rutas internas', () => {
		expect(safeRedirect('/tickets/3?x=1', '/inicio')).toBe('/tickets/3?x=1');
	});

	it('rechaza destinos externos o vacíos', () => {
		for (const target of [null, '', 'https://malo.com', '//malo.com', '/\\malo.com', 'tickets']) {
			expect(safeRedirect(target, '/inicio')).toBe('/inicio');
		}
	});
});
