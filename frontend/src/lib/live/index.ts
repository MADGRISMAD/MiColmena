import { writable } from 'svelte/store';
import { currentToken } from '#lib/stores/auth.js';
import { connectStream, type LiveEvent } from './stream';

export type { LiveEvent } from './stream';

/**
 * Último aviso recibido en tiempo real. Las páginas lo observan para recargar lo que muestran.
 * `seq` cambia en cada aviso, incluso si se repite el mismo ticket.
 */
export const liveEvent = writable<{ seq: number; event: LiveEvent | { type: 'reconnect' } }>({
	seq: 0,
	event: { type: 'reconnect' }
});

let seq = 0;

/** Conecta mientras la sesión esté abierta. Devuelve la función para desconectar. */
export function startLive(): () => void {
	let first = true;
	return connectStream({
		url: (import.meta.env.VITE_API_URL ?? '') + '/api/stream',
		getToken: currentToken,
		onEvent: (event) => liveEvent.set({ seq: ++seq, event }),
		onOpen: () => {
			// En la primera conexión no hace falta recargar nada.
			if (!first) liveEvent.set({ seq: ++seq, event: { type: 'reconnect' } });
			first = false;
		}
	});
}
