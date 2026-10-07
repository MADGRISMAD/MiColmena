import { writable } from 'svelte/store';
import { api, type OrgUsage } from '#lib/api/index.js';

/** La empresa del usuario, con lo que usa de su plan. null mientras carga. */
export const org = writable<OrgUsage | null>(null);

export async function loadOrg() {
	try {
		org.set(await api.getOrg());
	} catch {
		// La app sigue funcionando sin estos datos.
	}
}
