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
	active: boolean;
	/** Además de la campana, recibir avisos por correo. */
	email_notifications: boolean;
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
	tags: string[];
	custom_fields: Record<string, unknown>;
	created_at: string;
	updated_at: string;
	resolved_at: string | null;
	/** SLA: plazos calculados desde la creación según la prioridad. */
	first_response_at: string | null;
	first_response_due: string;
	resolution_due: string;
	satisfaction: Satisfaction;
	satisfaction_comment: string;
	rated_at: string | null;
}

export type Satisfaction = '' | 'good' | 'bad';

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
	sla_breached: number;
	satisfaction_30d: { good: number; bad: number };
}

export type EventKind =
	'created' | 'status' | 'priority' | 'assignee' | 'category' | 'title' | 'tags' | 'satisfaction';

export interface TicketEvent {
	id: number;
	actor: UserRef | null;
	kind: EventKind;
	old_value: string;
	new_value: string;
	created_at: string;
}

export interface Attachment {
	id: number;
	ticket_id: number;
	/** null = adjunto de la descripción del ticket. */
	comment_id: number | null;
	uploader: UserRef;
	filename: string;
	content_type: string;
	size: number;
	internal: boolean;
	created_at: string;
}

export interface Category {
	id: number;
	name: string;
	created_at: string;
}

export interface Macro {
	id: number;
	title: string;
	body: string;
	/** Estado al que pasa el ticket al usarla; vacío = no cambia. */
	status: TicketStatus | '';
	internal: boolean;
	created_at: string;
	updated_at: string;
}

export interface MacroInput {
	title: string;
	body: string;
	status: TicketStatus | '';
	internal: boolean;
}

export interface TagCount {
	name: string;
	count: number;
}

export interface SlaPolicy {
	priority: TicketPriority;
	first_response_minutes: number;
	resolution_minutes: number;
}

export type NotificationKind = 'new_ticket' | 'assigned' | 'comment' | 'mention' | 'status';

export interface Notification {
	id: number;
	ticket_id: number | null;
	actor: UserRef | null;
	kind: NotificationKind;
	summary: string;
	ticket_title: string;
	read_at: string | null;
	created_at: string;
}

export interface NotificationList {
	items: Notification[];
	unread: number;
}

export interface Article {
	id: number;
	title: string;
	body: string;
	category: string;
	published: boolean;
	author: UserRef | null;
	created_at: string;
	updated_at: string;
}

export interface ArticleInput {
	title: string;
	body: string;
	category: string;
	published: boolean;
}

export interface SavedView {
	id: number;
	name: string;
	/** Filtros de la lista, como "?priority=urgent&assignee=me". */
	query: string;
	created_at: string;
}

export interface Report {
	from: string;
	to: string;
	created: number;
	resolved: number;
	avg_first_response_hours: number | null;
	avg_resolution_hours: number | null;
	/** Proporción (0 a 1) de primeras respuestas dentro del SLA. */
	first_response_sla_met: number | null;
	satisfaction_good: number;
	satisfaction_bad: number;
	daily: { date: string; created: number; resolved: number }[];
	agents: {
		id: number;
		name: string;
		open_assigned: number;
		resolved: number;
		avg_resolution_hours: number | null;
		satisfaction_good: number;
		satisfaction_bad: number;
	}[];
	categories: { name: string; count: number }[];
}

export interface ReportRange {
	from?: string;
	to?: string;
	tz?: string;
}

export interface TicketFilters {
	status?: TicketStatus;
	priority?: TicketPriority;
	/** 'me', 'none' o el id de un agente. */
	assignee?: 'me' | 'none' | number;
	tag?: string;
	sla?: 'breached';
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
	tags?: string[];
}

export interface BulkUpdateInput {
	ids: number[];
	status?: TicketStatus;
	priority?: TicketPriority;
	assignee_id?: number | null;
	add_tags?: string[];
}

export interface CreateUserInput {
	name: string;
	email: string;
	role: Role;
	password: string;
}

export interface UpdateUserInput {
	name?: string;
	email?: string;
	role?: Role;
	active?: boolean;
	password?: string;
}
