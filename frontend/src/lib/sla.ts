import type { Ticket } from '#lib/api/types.js';

export interface SlaState {
	/** Plazo más cercano que sigue pendiente. */
	target: 'first_response' | 'resolution';
	due: Date;
	/** Milisegundos hasta el plazo (negativo si ya venció). */
	remaining: number;
	breached: boolean;
	/** Queda menos de una hora o menos del 25 % del plazo. */
	atRisk: boolean;
}

type SlaTicket = Pick<
	Ticket,
	'status' | 'created_at' | 'first_response_at' | 'first_response_due' | 'resolution_due'
>;

/** Estado del SLA de un ticket pendiente; null si ya está resuelto o cerrado. */
export function slaState(ticket: SlaTicket, now: Date = new Date()): SlaState | null {
	if (ticket.status === 'resolved' || ticket.status === 'closed') return null;
	const created = new Date(ticket.created_at).getTime();
	const targets: { target: SlaState['target']; due: number }[] = [];
	if (!ticket.first_response_at) {
		targets.push({ target: 'first_response', due: new Date(ticket.first_response_due).getTime() });
	}
	targets.push({ target: 'resolution', due: new Date(ticket.resolution_due).getTime() });
	const next = targets.reduce((a, b) => (b.due < a.due ? b : a));
	const remaining = next.due - now.getTime();
	const window = next.due - created;
	return {
		target: next.target,
		due: new Date(next.due),
		remaining,
		breached: remaining < 0,
		atRisk: remaining >= 0 && (remaining < 3_600_000 || remaining < window * 0.25)
	};
}

/** "45 min", "3 h 20 min", "2 días". */
export function formatDuration(ms: number): string {
	const minutes = Math.max(1, Math.round(Math.abs(ms) / 60_000));
	if (minutes < 60) return `${minutes} min`;
	const hours = Math.floor(minutes / 60);
	if (hours < 48) {
		const rest = minutes % 60;
		return rest ? `${hours} h ${rest} min` : `${hours} h`;
	}
	return `${Math.round(hours / 24)} días`;
}

/** Minutos a texto corto para los formularios de SLA: 90 → "1 h 30 min". */
export function formatMinutes(minutes: number): string {
	return formatDuration(minutes * 60_000);
}
