// Tipos que reflejan las respuestas JSON del backend en Go (backend/internal/api).

export const ROLES = ['admin', 'agent', 'customer'] as const;
export type Role = (typeof ROLES)[number];

export const TICKET_STATUSES = ['open', 'in_progress', 'waiting', 'resolved', 'closed'] as const;
export type TicketStatus = (typeof TICKET_STATUSES)[number];

export const TICKET_PRIORITIES = ['low', 'medium', 'high', 'urgent'] as const;
export type TicketPriority = (typeof TICKET_PRIORITIES)[number];

export interface User {
	id: number;
	name: string;
	email: string;
	role: Role;
	created_at: string;
}

export interface UserRef {
	id: number;
	name: string;
}

export interface AuthResponse {
	token: string;
	expires_at: string;
	user: User;
}

export interface Ticket {
	id: number;
	title: string;
	description: string;
	status: TicketStatus;
	priority: TicketPriority;
	category: string;
	requester: UserRef;
	assignee: UserRef | null;
	custom_fields: Record<string, unknown>;
	created_at: string;
	updated_at: string;
	resolved_at: string | null;
}

export interface Comment {
	id: number;
	ticket_id: number;
	author: UserRef;
	body: string;
	internal: boolean;
	created_at: string;
}

export interface Page<T> {
	items: T[];
	next_cursor: number | null;
}

export interface List<T> {
	items: T[];
}

export interface Stats {
	by_status: Record<TicketStatus, number>;
	open_by_priority: Record<TicketPriority, number>;
	unassigned_open: number;
	avg_resolution_hours_30d: number | null;
}

export interface TicketFilters {
	status?: TicketStatus;
	priority?: TicketPriority;
	/** 'me', 'none' o el id de un agente. */
	assignee?: 'me' | 'none' | number;
	q?: string;
	before?: number;
	limit?: number;
}

export interface CreateTicketInput {
	title: string;
	description?: string;
	priority?: TicketPriority;
	category?: string;
	custom_fields?: Record<string, unknown>;
	requester_id?: number;
}

export interface UpdateTicketInput {
	title?: string;
	description?: string;
	status?: TicketStatus;
	priority?: TicketPriority;
	category?: string;
	custom_fields?: Record<string, unknown>;
	/** null desasigna el ticket. */
	assignee_id?: number | null;
}
