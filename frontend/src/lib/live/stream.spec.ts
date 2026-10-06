import { describe, expect, it, vi } from 'vitest';
import { connectStream, parseSse } from './stream';

describe('parseSse', () => {
	it('separa eventos y conserva el trozo incompleto', () => {
		const { events, rest } = parseSse('retry: 5000\n\n: ping\n\ndata: {"a":1}\n\ndata: {"b"');
		expect(events).toEqual(['{"a":1}']);
		expect(rest).toBe('data: {"b"');
	});

	it('admite saltos de línea CRLF', () => {
		expect(parseSse('data: x\r\n\r\n').events).toEqual(['x']);
	});
});

describe('connectStream', () => {
	it('envía el token en la cabecera y entrega los eventos', async () => {
		const body = new ReadableStream({
			start(controller) {
				const enc = new TextEncoder();
				controller.enqueue(
					enc.encode('data: {"type":"ticket","ticket_id":3}\n\ndata: {"type":"noti')
				);
				controller.enqueue(enc.encode('fication"}\n\n'));
				controller.close();
			}
		});
		const fetch = vi.fn().mockResolvedValueOnce(new Response(body, { status: 200 }));
		const events: unknown[] = [];
		const stop = connectStream({
			url: '/api/stream',
			getToken: () => 'tok',
			onEvent: (e) => events.push(e),
			fetch
		});
		await vi.waitFor(() => expect(events).toHaveLength(2));
		stop();

		expect(fetch.mock.calls[0][1].headers.Authorization).toBe('Bearer tok');
		expect(events).toEqual([{ type: 'ticket', ticket_id: 3 }, { type: 'notification' }]);
	});

	it('no se conecta sin sesión', () => {
		const fetch = vi.fn();
		connectStream({ url: '/x', getToken: () => null, onEvent: () => {}, fetch })();
		expect(fetch).not.toHaveBeenCalled();
	});
});
