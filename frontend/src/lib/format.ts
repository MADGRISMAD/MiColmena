import type { Role, TicketPriority, TicketStatus } from '#lib/api/types.js';

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
