import { describe, expect, it } from 'vitest';
import { fillMacro, mentionQuery } from './macros';

describe('fillMacro', () => {
	it('reemplaza las variables con el nombre de pila', () => {
		expect(
			fillMacro('Hola {{solicitante}}, soy {{agente}}. Tu caso {{ticket}}…', {
				solicitante: 'Ana López',
				agente: 'Luis Ramírez',
				ticket: 12
			})
		).toBe('Hola Ana, soy Luis. Tu caso #12…');
	});
});

describe('mentionQuery', () => {
	it('detecta una mención en curso', () => {
		expect(mentionQuery('Hola @lu')).toBe('lu');
		expect(mentionQuery('@')).toBe('');
		expect(mentionQuery('@Már')).toBe('Már');
	});

	it('ignora emails y texto sin arroba', () => {
		expect(mentionQuery('escribe a ana@correo')).toBeNull();
		expect(mentionQuery('hola')).toBeNull();
		expect(mentionQuery('@luis ')).toBeNull();
	});
});
