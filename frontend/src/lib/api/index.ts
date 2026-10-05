import { clearSession, currentToken } from '#lib/stores/auth.js';
import { createApiClient } from './client';

export { ApiError, createApiClient, toQuery, type ApiClient } from './client';
export * from './types';

/**
 * Cliente de la API para toda la aplicación.
 * En desarrollo, Vite redirige /api al backend (ver vite.config.ts).
 * En producción, define VITE_API_URL con la URL del backend, por ejemplo https://api.micolmena.com.
 */
export const api = createApiClient({
	baseUrl: import.meta.env.VITE_API_URL ?? '',
	getToken: currentToken,
	onUnauthorized: clearSession
});
