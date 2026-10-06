import { describe, expect, it } from 'vitest';
import { parseTicketFilters } from './filters';

describe('parseTicketFilters', () => {
	it('lee los filtros válidos de la URL', () => {
		const params = new URLSearchParams('status=open&priority=urgent&assignee=12&q=+impresora+');
		expect(parseTicketFilters(params)).toEqual({
			status: 'open',
			priority: 'urgent',
			assignee: 12,
			q: 'impresora'
		});
	});

	it('acepta los valores especiales de asignación', () => {
		expect(parseTicketFilters(new URLSearchParams('assignee=me'))).toEqual({ assignee: 'me' });
		expect(parseTicketFilters(new URLSearchParams('assignee=none'))).toEqual({ assignee: 'none' });
	});

	it('lee la etiqueta y el filtro de SLA', () => {
		expect(parseTicketFilters(new URLSearchParams('tag=VIP&sla=breached'))).toEqual({
			tag: 'vip',
			sla: 'breached'
		});
	});

	it('descarta valores inválidos', () => {
		const params = new URLSearchParams(
			'status=borrado&priority=x&assignee=abc&q=%20&sla=x&tag=%20'
		);
		expect(parseTicketFilters(params)).toEqual({});
	});
});
