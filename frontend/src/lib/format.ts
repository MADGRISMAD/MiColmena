import type {
	AssetCategory,
	AssetEvent,
	AssetState,
	Role,
	TicketEvent,
	TicketPriority,
	TicketStatus
} from '#lib/api/types.js';

export const statusLabels: Record<TicketStatus, string> = {
	open: 'Abierto',
	in_progress: 'En curso',
	waiting: 'En espera',
	resolved: 'Resuelto',
	closed: 'Cerrado'
};

export const priorityLabels: Record<TicketPriority, string> = {
	low: 'Baja',
	medium: 'Media',
	high: 'Alta',
	urgent: 'Urgente'
};

export const roleLabels: Record<Role, string> = {
	admin: 'Administrador',
	agent: 'Agente',
	customer: 'Cliente'
};

const dateTime = new Intl.DateTimeFormat('es', { dateStyle: 'medium', timeStyle: 'short' });
const relative = new Intl.RelativeTimeFormat('es', { numeric: 'auto' });

export function formatDateTime(iso: string): string {
	return dateTime.format(new Date(iso));
}

const units: [Intl.RelativeTimeFormatUnit, number][] = [
	['year', 365 * 24 * 3600],
	['month', 30 * 24 * 3600],
	['week', 7 * 24 * 3600],
	['day', 24 * 3600],
	['hour', 3600],
	['minute', 60]
];

/** "hace 5 minutos", "ayer"… */
export function formatRelative(iso: string, now: Date = new Date()): string {
	const seconds = (new Date(iso).getTime() - now.getTime()) / 1000;
	for (const [unit, size] of units) {
		if (Math.abs(seconds) >= size) return relative.format(Math.round(seconds / size), unit);
	}
	return 'ahora mismo';
}

/** Duración en horas, legible: "45 min", "3,5 h", "2 días". */
export function formatHours(hours: number | null): string {
	if (hours === null) return '—';
	if (hours < 1) return `${Math.max(1, Math.round(hours * 60))} min`;
	if (hours < 48) return `${hours.toLocaleString('es', { maximumFractionDigits: 1 })} h`;
	return `${Math.round(hours / 24)} días`;
}

const isStatus = (v: string): v is TicketStatus => v in statusLabels;
const isPriority = (v: string): v is TicketPriority => v in priorityLabels;

/** Describe un cambio del historial: "cambió el estado de Abierto a En espera". */
export function describeEvent(e: Pick<TicketEvent, 'kind' | 'old_value' | 'new_value'>): string {
	const status = (v: string) => (isStatus(v) ? statusLabels[v] : v);
	const priority = (v: string) => (isPriority(v) ? priorityLabels[v] : v);
	switch (e.kind) {
		case 'created':
			return 'abrió el ticket';
		case 'status':
			return `cambió el estado de ${status(e.old_value)} a ${status(e.new_value)}`;
		case 'priority':
			return `cambió la prioridad de ${priority(e.old_value)} a ${priority(e.new_value)}`;
		case 'assignee':
			if (!e.new_value) return `quitó la asignación de ${e.old_value}`;
			if (!e.old_value) return `asignó el ticket a ${e.new_value}`;
			return `reasignó el ticket de ${e.old_value} a ${e.new_value}`;
		case 'category':
			return e.new_value ? `cambió la categoría a ${e.new_value}` : 'quitó la categoría';
		case 'title':
			return `cambió el título a «${e.new_value}»`;
		case 'tags': {
			const before = e.old_value ? e.old_value.split(',') : [];
			const after = e.new_value ? e.new_value.split(',') : [];
			const added = after.filter((t) => !before.includes(t)).map((t) => `#${t}`);
			const removed = before.filter((t) => !after.includes(t)).map((t) => `#${t}`);
			return [
				added.length ? `añadió ${added.join(', ')}` : '',
				removed.length ? `quitó ${removed.join(', ')}` : ''
			]
				.filter(Boolean)
				.join(' y ');
		}
		case 'satisfaction':
			return `valoró la atención como ${e.new_value === 'good' ? 'buena' : 'mala'}`;
		default:
			return 'hizo un cambio';
	}
}

// --- Activos ---

export const assetCategoryLabels: Record<AssetCategory, string> = {
	computer: 'Computadora',
	phone: 'Teléfono',
	monitor: 'Monitor',
	network: 'Red',
	peripheral: 'Periférico',
	software: 'Software',
	other: 'Otro'
};

export const assetStateLabels: Record<AssetState, string> = {
	in_stock: 'En almacén',
	in_use: 'En uso',
	in_repair: 'En reparación',
	retired: 'Retirado'
};

/** Nombre de cada campo editable, tal como llega en el evento 'updated' del equipo. */
export const assetFieldLabels: Record<string, string> = {
	tag: 'etiqueta',
	name: 'nombre',
	category: 'categoría',
	model: 'modelo',
	serial: 'serie',
	purchase_date: 'fecha de compra',
	warranty_until: 'garantía',
	purchase_cost: 'costo',
	notes: 'notas'
};

const isAssetState = (v: string): v is AssetState => v in assetStateLabels;

/** Texto de un evento del historial de un equipo: "Estado: En almacén → En uso". */
export function describeAssetEvent(
	e: Pick<AssetEvent, 'kind' | 'old_value' | 'new_value'>
): string {
	const state = (v: string) => (isAssetState(v) ? assetStateLabels[v] : v);
	switch (e.kind) {
		case 'created':
			return 'Se dio de alta el equipo';
		case 'state':
			return `Estado: ${state(e.old_value)} → ${state(e.new_value)}`;
		case 'assigned':
			return e.new_value
				? `Asignado a ${e.new_value}`
				: `Se quitó el responsable${e.old_value ? ` (era ${e.old_value})` : ''}`;
		case 'location':
			return e.new_value
				? `Ubicación: ${e.old_value || 'sin ubicación'} → ${e.new_value}`
				: `Se quitó la ubicación (era ${e.old_value})`;
		case 'updated': {
			const fields = e.new_value
				.split(',')
				.map((f) => assetFieldLabels[f.trim()] ?? f.trim())
				.filter(Boolean);
			return `Se actualizaron: ${fields.join(', ')}`;
		}
		case 'ticket':
			return `Vinculado al ticket ${e.new_value || e.old_value}`;
		default:
			return 'Se hizo un cambio';
	}
}

/** Estado de la garantía: vencida, por vencer (60 días) o vigente. null si no hay fecha. */
export function warrantyStatus(
	warrantyUntil: string | null,
	now: Date = new Date()
): 'expired' | 'expiring' | 'ok' | null {
	if (!warrantyUntil) return null;
	const today = now.toISOString().slice(0, 10);
	if (warrantyUntil < today) return 'expired';
	const soon = new Date(now.getTime() + 60 * 86_400_000).toISOString().slice(0, 10);
	return warrantyUntil <= soon ? 'expiring' : 'ok';
}

/** "2024-01-10" -> "10 ene 2024" (sin pasar por zonas horarias). */
export function formatDate(date: string | null): string {
	if (!date) return '—';
	const [y, m, d] = date.split('-').map(Number);
	return new Intl.DateTimeFormat('es', { dateStyle: 'medium' }).format(new Date(y, m - 1, d));
}
