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
	/** Administrador permanente: no se desactiva, ni se le cambia el rol o el email. */
	permanent?: boolean;
	/** Empresa a la que pertenece la cuenta. */
	org_id: number;
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
	/** Incidente (algo falló) o solicitud (se pidió algo del catálogo de servicios). */
	kind: TicketKind;
	catalog_item: { id: number; name: string } | null;
}

export const TICKET_KINDS = ['incident', 'request'] as const;
export type TicketKind = (typeof TICKET_KINDS)[number];

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
	/** Sin resolver y asignados a quien consulta. */
	mine_open: number;
	/** Solicitudes del catálogo sin resolver. */
	requests_open: number;
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

export type NotificationKind =
	'new_ticket' | 'assigned' | 'comment' | 'mention' | 'status' | 'lead';

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
	/** Empresa dueña del artículo (cada empresa tiene su centro de ayuda). */
	org: OrgChoice;
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
	kind?: TicketKind;
	/** Id de un activo: tickets vinculados a ese equipo. */
	asset?: number;
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
	/** Equipo del que trata el ticket (un cliente solo puede elegir los suyos). */
	asset_id?: number;
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

export const LEAD_TEAM_SIZES = ['1-3', '4-10', '11-25', '26+'] as const;
export const LEAD_PLANS = ['emprendedor', 'profesional', 'empresa'] as const;
export type LeadPlan = (typeof LEAD_PLANS)[number];

export interface LeadInput {
	name: string;
	company: string;
	email: string;
	phone: string;
	/** Lo elegido en la calculadora de precios; 0 = no lo indicó. */
	agents: number;
	people: number;
	message: string;
	/** Campo trampa para bots: siempre vacío. */
	website?: string;
}

export interface Lead extends Omit<LeadInput, 'website'> {
	id: number;
	/** De solicitudes anteriores a la calculadora. */
	team_size: (typeof LEAD_TEAM_SIZES)[number] | '';
	plan: LeadPlan | '';
	handled: boolean;
	created_at: string;
}

/** Una empresa entre las que el email tiene cuenta (para elegir al iniciar sesión). */
export interface OrgChoice {
	slug: string;
	name: string;
}

export interface Organization {
	id: number;
	name: string;
	/** Dirección del portal: /e/<slug>. */
	slug: string;
	/** Personas en la empresa (definen el precio). */
	people: number;
	/** Agentes permitidos por el plan; null = sin límite. */
	max_agents: number | null;
	suspended: boolean;
	created_at: string;
}

export interface OrgUsage extends Organization {
	/** Agentes y administradores activos. */
	agents: number;
	/** La empresa de la propia plataforma (sin plan ni límites). */
	platform: boolean;
}

export interface PlatformOrg extends Organization {
	agents: number;
	customers: number;
	tickets: number;
	last_ticket_at: string | null;
}

export interface SignupInput {
	company: string;
	people: number;
	name: string;
	email: string;
	password: string;
	accept_terms: boolean;
}

// --- Catálogo de servicios ---

export const FIELD_TYPES = ['text', 'textarea', 'select', 'number', 'date'] as const;
export type FieldType = (typeof FIELD_TYPES)[number];

export const CATALOG_ICONS = [
	'package',
	'shield',
	'mail',
	'phone',
	'users',
	'wrench',
	'laptop',
	'key',
	'chart'
] as const;
export type CatalogIcon = (typeof CATALOG_ICONS)[number];

/** Una pregunta del formulario de un servicio. */
export interface FormField {
	key: string;
	label: string;
	type: FieldType;
	required: boolean;
	options?: string[];
}

export interface ServiceItem {
	id: number;
	category_id: number;
	name: string;
	description: string;
	fields: FormField[];
	priority: TicketPriority;
	ticket_category: string;
	active: boolean;
	position: number;
	created_at: string;
}

export interface ServiceCategory {
	id: number;
	name: string;
	description: string;
	icon: CatalogIcon;
	position: number;
	items: ServiceItem[];
}

export interface ServiceCategoryInput {
	name: string;
	description?: string;
	icon?: CatalogIcon;
	position?: number;
}

export interface ServiceItemInput {
	category_id: number;
	name: string;
	description?: string;
	fields?: (Omit<FormField, 'key'> & { key?: string })[];
	priority?: TicketPriority;
	ticket_category?: string;
	active?: boolean;
	position?: number;
}

export interface ServiceRequestInput {
	/** Respuestas del formulario, por el `key` de cada campo. */
	values: Record<string, string | number>;
	notes?: string;
	asset_id?: number;
}

// --- Activos ---

export const ASSET_CATEGORIES = [
	'computer',
	'phone',
	'monitor',
	'network',
	'peripheral',
	'software',
	'other'
] as const;
export type AssetCategory = (typeof ASSET_CATEGORIES)[number];

export const ASSET_STATES = ['in_stock', 'in_use', 'in_repair', 'retired'] as const;
export type AssetState = (typeof ASSET_STATES)[number];

export interface Asset {
	id: number;
	tag: string;
	name: string;
	category: AssetCategory;
	model: string;
	serial: string;
	state: AssetState;
	assigned_to: UserRef | null;
	location: string;
	/** Fechas AAAA-MM-DD. */
	purchase_date: string | null;
	purchase_cost: number | null;
	warranty_until: string | null;
	notes: string;
	created_at: string;
	updated_at: string;
	/** Tickets sin resolver vinculados a este equipo. */
	open_tickets: number;
}

/** Datos editables de un equipo (el PATCH reemplaza todos). */
export interface AssetInput {
	tag: string;
	name: string;
	category?: AssetCategory;
	model?: string;
	serial?: string;
	state?: AssetState;
	assigned_to?: number | null;
	location?: string;
	purchase_date?: string;
	purchase_cost?: number | null;
	warranty_until?: string;
	notes?: string;
}

export type AssetEventKind = 'created' | 'state' | 'assigned' | 'location' | 'updated' | 'ticket';

export interface AssetEvent {
	id: number;
	actor: UserRef | null;
	kind: AssetEventKind;
	/** Para 'updated', new_value son las claves de los campos cambiados separadas por comas. */
	old_value: string;
	new_value: string;
	created_at: string;
}

export interface AssetTicket {
	id: number;
	title: string;
	status: TicketStatus;
	priority: TicketPriority;
	kind: TicketKind;
	created_at: string;
}

export interface AssetDetail {
	asset: Asset;
	events: AssetEvent[];
	tickets: AssetTicket[];
}

export interface AssetFilters {
	q?: string;
	state?: AssetState;
	category?: AssetCategory;
	/** 'me', 'none' o el id de un usuario. */
	assigned?: 'me' | 'none' | number;
	warranty?: 'expiring' | 'expired';
	before?: number;
	limit?: number;
}

export interface AssetSummary {
	total: number;
	total_value: number;
	by_state: Record<AssetState, number>;
	by_category: Record<AssetCategory, number>;
	warranty_expiring: number;
	warranty_expired: number;
	with_open_tickets: number;
	unassigned_in_use: number;
}
