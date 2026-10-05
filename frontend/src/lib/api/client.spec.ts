import { describe, expect, it, vi } from 'vitest';
import { ApiError, createApiClient, toQuery } from './client';

function jsonResponse(status: number, body: unknown) {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

describe('toQuery', () => {
	it('omite valores vacíos', () => {
		expect(toQuery({ status: 'open', q: '', before: undefined, limit: 10, x: null })).toBe(
			'?status=open&limit=10'
		);
	});

	it('devuelve cadena vacía sin parámetros', () => {
		expect(toQuery({})).toBe('');
	});
});

describe('createApiClient', () => {
	it('envía el token y el cuerpo JSON', async () => {
		const fetch = vi.fn().mockResolvedValue(jsonResponse(201, { id: 7 }));
		const api = createApiClient({ baseUrl: 'https://api.test', getToken: () => 'abc', fetch });

		await api.createTicket({ title: 'Hola' });

		expect(fetch).toHaveBeenCalledWith('https://api.test/api/tickets', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json', Authorization: 'Bearer abc' },
			body: JSON.stringify({ title: 'Hola' })
		});
	});

	it('no envía Authorization sin sesión ni Content-Type sin cuerpo', async () => {
		const fetch = vi.fn().mockResolvedValue(jsonResponse(200, { items: [], next_cursor: null }));
		const api = createApiClient({ fetch });

		await api.listTickets({ status: 'open', assignee: 'me' });

		const [url, init] = fetch.mock.calls[0];
		expect(url).toBe('/api/tickets?status=open&assignee=me');
		expect(init.headers).toEqual({});
	});

	it('convierte los errores de validación en ApiError con campos', async () => {
		const fetch = vi
			.fn()
			.mockResolvedValue(
				jsonResponse(422, { error: 'datos inválidos', fields: { email: 'no es un email válido' } })
			);
		const api = createApiClient({ fetch });

		const err = await api.register('Ana', 'x', '12345678').catch((e) => e);

		expect(err).toBeInstanceOf(ApiError);
		expect(err.status).toBe(422);
		expect(err.fields).toEqual({ email: 'no es un email válido' });
	});

	it('avisa de 401 para cerrar la sesión', async () => {
		const onUnauthorized = vi.fn();
		const fetch = vi.fn().mockResolvedValue(jsonResponse(401, { error: 'token inválido' }));
		const api = createApiClient({ fetch, onUnauthorized });

		await expect(api.me()).rejects.toThrow('token inválido');
		expect(onUnauthorized).toHaveBeenCalledOnce();
	});

	it('traduce un fallo de red a un mensaje legible', async () => {
		const fetch = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'));
		const api = createApiClient({ fetch });

		const err = await api.health().catch((e) => e);

		expect(err).toBeInstanceOf(ApiError);
		expect(err.status).toBe(0);
		expect(err.message).toBe('No se pudo conectar con el servidor');
	});
});
