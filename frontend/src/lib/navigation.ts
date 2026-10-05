import type { User } from '#lib/api/types.js';

/**
 * Devuelve la ruta a la que volver tras iniciar sesión. Solo acepta rutas internas
 * para que un enlace como /login?redirect=https://otro-sitio.com no saque al usuario de la app.
 */
export function safeRedirect(target: string | null, fallback: string): string {
	if (!target || !target.startsWith('/') || target.startsWith('//') || target.startsWith('/\\')) {
		return fallback;
	}
	return target;
}

/** Página de inicio según el rol: el equipo va al dashboard y los clientes a sus tickets. */
export function homeFor(user: User): '/(app)/dashboard' | '/(app)/tickets' {
	return user.role === 'customer' ? '/(app)/tickets' : '/(app)/dashboard';
}
