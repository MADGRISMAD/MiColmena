import { describe, expect, it } from 'vitest';
import { formatHours, formatRelative } from './format';

describe('formatRelative', () => {
	const now = new Date('2026-10-05T12:00:00Z');

	it('usa la unidad más grande que cabe', () => {
		expect(formatRelative('2026-10-05T11:55:00Z', now)).toBe('hace 5 minutos');
		expect(formatRelative('2026-10-04T12:00:00Z', now)).toBe('ayer');
		expect(formatRelative('2026-09-21T12:00:00Z', now)).toBe('hace 2 semanas');
	});

	it('dice "ahora mismo" por debajo de un minuto', () => {
		expect(formatRelative('2026-10-05T11:59:30Z', now)).toBe('ahora mismo');
	});
});

describe('formatHours', () => {
	it('elige minutos, horas o días', () => {
		expect(formatHours(null)).toBe('—');
		expect(formatHours(0.25)).toBe('15 min');
		expect(formatHours(3.5)).toBe('3,5 h');
		expect(formatHours(72)).toBe('3 días');
	});
});
