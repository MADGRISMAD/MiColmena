import { describe, expect, it } from 'vitest';
import { formatDuration, slaState } from './sla';

const base = {
	status: 'open' as const,
	created_at: '2026-01-01T10:00:00Z',
	first_response_at: null,
	first_response_due: '2026-01-01T11:00:00Z',
	resolution_due: '2026-01-01T14:00:00Z'
};

describe('slaState', () => {
	it('mira primero la primera respuesta pendiente', () => {
		const s = slaState(base, new Date('2026-01-01T10:20:00Z'))!;
		expect(s.target).toBe('first_response');
		expect(s.breached).toBe(false);
		expect(s.atRisk).toBe(true); // quedan 40 min
	});

	it('con primera respuesta, pasa a la resolución', () => {
		const s = slaState(
			{ ...base, first_response_at: '2026-01-01T10:30:00Z' },
			new Date('2026-01-01T11:30:00Z')
		)!;
		expect(s.target).toBe('resolution');
		expect(s.atRisk).toBe(false);
	});

	it('marca el vencimiento', () => {
		const s = slaState(base, new Date('2026-01-01T12:00:00Z'))!;
		expect(s.breached).toBe(true);
		expect(s.remaining).toBeLessThan(0);
	});

	it('no aplica a tickets resueltos', () => {
		expect(slaState({ ...base, status: 'resolved' })).toBeNull();
	});
});

describe('formatDuration', () => {
	it('elige la unidad', () => {
		expect(formatDuration(30 * 60_000)).toBe('30 min');
		expect(formatDuration(200 * 60_000)).toBe('3 h 20 min');
		expect(formatDuration(-120 * 60_000)).toBe('2 h');
		expect(formatDuration(72 * 3_600_000)).toBe('3 días');
	});
});
