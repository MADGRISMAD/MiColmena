import { describe, expect, it } from 'vitest';
import { describeEvent, formatHours, formatRelative } from './format';

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

describe('describeEvent', () => {
	it('describe cada tipo de cambio', () => {
		expect(describeEvent({ kind: 'status', old_value: 'open', new_value: 'waiting' })).toBe(
			'cambió el estado de Abierto a En espera'
		);
		expect(describeEvent({ kind: 'assignee', old_value: '', new_value: 'Luis' })).toBe(
			'asignó el ticket a Luis'
		);
		expect(describeEvent({ kind: 'assignee', old_value: 'Luis', new_value: '' })).toBe(
			'quitó la asignación de Luis'
		);
		expect(describeEvent({ kind: 'tags', old_value: 'a,b', new_value: 'b,c' })).toBe(
			'añadió #c y quitó #a'
		);
	});
});
