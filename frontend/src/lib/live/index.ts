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

type Listener = (event: LiveEvent | { type: 'reconnect' }) => void;
const listeners = new Set<Listener>();

/**
 * Llama a `fn` con cada aviso, uno por uno. A diferencia del store (que puede agrupar
 * dos avisos seguidos en una sola actualización), aquí no se pierde ninguno.
 * Devuelve la función para dejar de escuchar; se puede devolver tal cual desde un $effect.
 */
export function onLive(fn: Listener): () => void {
	listeners.add(fn);
	return () => listeners.delete(fn);
}

function emit(event: LiveEvent | { type: 'reconnect' }) {
	liveEvent.set({ seq: ++seq, event });
	for (const fn of listeners) fn(event);
}

/** Conecta mientras la sesión esté abierta. Devuelve la función para desconectar. */
export function startLive(): () => void {
	return connectStream({
		url: (import.meta.env.VITE_API_URL ?? '') + '/api/stream',
		getToken: currentToken,
		onEvent: emit,
		// Al (re)conectar se recarga: un cambio ocurrido mientras se abría la conexión no se pierde.
		onOpen: () => emit({ type: 'reconnect' })
	});
}

/** Pide a la campana que se recargue (por ejemplo, tras marcar notificaciones como leídas). */
export const notificationsVersion = writable(0);
export function refreshNotifications() {
	notificationsVersion.update((n) => n + 1);
}
