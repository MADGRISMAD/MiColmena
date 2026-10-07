import { derived, get, writable } from 'svelte/store';
import type { User } from '#lib/api/types.js';

const TOKEN_KEY = 'micolmena_token';

function readToken(): string | null {
	try {
		return localStorage.getItem(TOKEN_KEY);
	} catch {
		return null;
	}
}

/** Token de sesión, guardado en localStorage para sobrevivir a recargas. */
export const token = writable<string | null>(typeof window === 'undefined' ? null : readToken());

token.subscribe((value) => {
	if (typeof window === 'undefined') return;
	try {
		if (value) localStorage.setItem(TOKEN_KEY, value);
		else localStorage.removeItem(TOKEN_KEY);
	} catch {
		// Sin almacenamiento (modo privado): la sesión dura lo que la pestaña.
	}
});

/** Usuario actual; null mientras no se ha cargado o si no hay sesión. */
export const user = writable<User | null>(null);

export const isAuthenticated = derived(token, ($token) => $token !== null);

/** Agentes y administradores ven todos los tickets y el dashboard. */
export const isStaff = derived(user, ($user) => $user?.role === 'agent' || $user?.role === 'admin');

export function setSession(newToken: string, newUser: User) {
	token.set(newToken);
	user.set(newUser);
}

export function clearSession() {
	token.set(null);
	user.set(null);
}

export function currentToken(): string | null {
	return get(token);
}

const LOGOUT_REASON_KEY = 'micolmena_logout_reason';

/** Cierra la sesión porque la API la rechazó, y guarda el motivo para mostrarlo en el login. */
export function sessionExpired(message: string) {
	if (get(token)) {
		try {
			sessionStorage.setItem(LOGOUT_REASON_KEY, message);
		} catch {
			// Sin almacenamiento: el login simplemente no muestra el motivo.
		}
	}
	clearSession();
}

/** Devuelve (y olvida) el motivo por el que se cerró la última sesión. */
export function takeLogoutReason(): string | null {
	try {
		const reason = sessionStorage.getItem(LOGOUT_REASON_KEY);
		sessionStorage.removeItem(LOGOUT_REASON_KEY);
		return reason;
	} catch {
		return null;
	}
}
