import { describe, expect, it } from 'vitest';
import { commentSchema, registerSchema, ticketSchema } from './schemas';

function fieldErrors(result: { success: boolean; error?: { issues: { path: PropertyKey[] }[] } }) {
	return result.error?.issues.map((i) => i.path.join('.')) ?? [];
}

describe('registerSchema', () => {
	const valid = { name: 'Ana', email: 'ana@x.com', password: 'secreto123', confirm: 'secreto123' };

	it('acepta datos válidos', () => {
		expect(registerSchema.safeParse(valid).success).toBe(true);
	});

	it('exige que las contraseñas coincidan', () => {
		const result = registerSchema.safeParse({ ...valid, confirm: 'otra' });
		expect(fieldErrors(result)).toEqual(['confirm']);
	});

	it('valida email, nombre y longitud de contraseña', () => {
		const result = registerSchema.safeParse({
			name: '  ',
			email: 'x',
			password: '1',
			confirm: '1'
		});
		expect(fieldErrors(result).sort()).toEqual(['email', 'name', 'password']);
	});
});

describe('ticketSchema', () => {
	it('recorta el título y rechaza uno vacío', () => {
		const ok = ticketSchema.safeParse({
			title: '  Hola  ',
			description: '',
			priority: 'high',
			category: ''
		});
		expect(ok.success && ok.data.title).toBe('Hola');

		const empty = ticketSchema.safeParse({
			title: '   ',
			description: '',
			priority: 'high',
			category: ''
		});
		expect(fieldErrors(empty)).toEqual(['title']);
	});

	it('rechaza prioridades que no existen', () => {
		const result = ticketSchema.safeParse({
			title: 'x',
			description: '',
			priority: 'altísima',
			category: ''
		});
		expect(fieldErrors(result)).toEqual(['priority']);
	});
});

describe('commentSchema', () => {
	it('rechaza comentarios en blanco', () => {
		expect(commentSchema.safeParse({ body: '  \n ', internal: false }).success).toBe(false);
	});
});
