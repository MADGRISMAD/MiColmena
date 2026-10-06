import { writable } from 'svelte/store';
import { api, type SavedView } from '#lib/api/index.js';

/** Vistas guardadas del usuario, compartidas entre la barra lateral y la lista de tickets. */
export const savedViews = writable<SavedView[]>([]);

export async function loadViews() {
	try {
		savedViews.set((await api.listViews()).items);
	} catch {
		savedViews.set([]);
	}
}

export async function saveView(name: string, query: string) {
	const view = await api.createView(name, query);
	savedViews.update((list) => [...list, view]);
	return view;
}

export async function removeView(id: number) {
	await api.deleteView(id);
	savedViews.update((list) => list.filter((v) => v.id !== id));
}
