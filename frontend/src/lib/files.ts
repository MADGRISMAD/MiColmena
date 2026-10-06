import { api, type Attachment } from '#lib/api/index.js';

/** "820 B", "1,2 MB". */
export function formatBytes(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`;
	const units = ['KB', 'MB', 'GB'];
	let value = bytes / 1024;
	let unit = 0;
	while (value >= 1024 && unit < units.length - 1) {
		value /= 1024;
		unit++;
	}
	return `${value.toLocaleString('es', { maximumFractionDigits: 1 })} ${units[unit]}`;
}

export const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

export function isImage(a: Pick<Attachment, 'content_type'>): boolean {
	return a.content_type.startsWith('image/');
}

/**
 * Descarga el adjunto con el token de la sesión y lo guarda con su nombre.
 * Los enlaces normales no sirven: la API pide la cabecera Authorization.
 */
export async function saveAttachment(a: Attachment) {
	const blob = await api.downloadAttachment(a.id);
	const url = URL.createObjectURL(blob);
	const link = document.createElement('a');
	link.href = url;
	link.download = a.filename;
	document.body.append(link);
	link.click();
	link.remove();
	setTimeout(() => URL.revokeObjectURL(url), 10_000);
}
