// Conexión en tiempo real con la API (Server-Sent Events). Se usa fetch en vez de EventSource
// porque EventSource no permite enviar la cabecera Authorization, y el token no debe ir en la URL.

export type LiveEvent = { type: 'ticket'; ticket_id: number } | { type: 'notification' };

export interface StreamOptions {
	url: string;
	getToken: () => string | null;
	onEvent: (event: LiveEvent) => void;
	/** Se llama al (re)conectar: conviene recargar datos por si se perdió algún aviso. */
	onOpen?: () => void;
	fetch?: typeof fetch;
}

/** Separa el texto recibido en eventos SSE y devuelve los datos de cada uno, más lo que sobra. */
export function parseSse(buffer: string): { events: string[]; rest: string } {
	const events: string[] = [];
	const blocks = buffer.replace(/\r\n/g, '\n').split('\n\n');
	const rest = blocks.pop() ?? '';
	for (const block of blocks) {
		const data = block
			.split('\n')
			.filter((line) => line.startsWith('data:'))
			.map((line) => line.slice(5).trimStart())
			.join('\n');
		if (data) events.push(data);
	}
	return { events, rest };
}

/** Abre la conexión y reconecta con espera creciente si se corta. Devuelve una función para cerrarla. */
export function connectStream(options: StreamOptions): () => void {
	const doFetch = options.fetch ?? ((...args: Parameters<typeof fetch>) => fetch(...args));
	let controller: AbortController | null = null;
	let stopped = false;
	let attempt = 0;
	let timer: ReturnType<typeof setTimeout> | undefined;

	async function run() {
		const token = options.getToken();
		if (stopped || !token) return;
		controller = new AbortController();
		try {
			const res = await doFetch(options.url, {
				headers: { Authorization: `Bearer ${token}`, Accept: 'text/event-stream' },
				signal: controller.signal
			});
			if (!res.ok || !res.body) throw new Error(`stream ${res.status}`);
			attempt = 0;
			options.onOpen?.();

			const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
			let buffer = '';
			for (;;) {
				const { value, done } = await reader.read();
				if (done) break;
				const parsed = parseSse(buffer + value);
				buffer = parsed.rest;
				for (const data of parsed.events) {
					try {
						options.onEvent(JSON.parse(data) as LiveEvent);
					} catch {
						// Un evento mal formado no debe cortar la conexión.
					}
				}
			}
		} catch {
			// Se reintenta abajo.
		}
		if (stopped) return;
		attempt++;
		const delay = Math.min(30_000, 1000 * 2 ** Math.min(attempt, 5));
		timer = setTimeout(run, delay);
	}

	run();
	return () => {
		stopped = true;
		clearTimeout(timer);
		controller?.abort();
	};
}
