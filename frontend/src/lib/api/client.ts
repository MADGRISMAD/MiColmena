import type {
	Article,
	ArticleInput,
	Asset,
	AssetDetail,
	AssetFilters,
	AssetInput,
	AssetSummary,
	Attachment,
	AuthResponse,
	BulkUpdateInput,
	Category,
	Comment,
	CreateTicketInput,
	CreateUserInput,
	List,
	Lead,
	LeadInput,
	Macro,
	MacroInput,
	NotificationList,
	OrgChoice,
	OrgUsage,
	Organization,
	PlatformOrg,
	SignupInput,
	Page,
	Report,
	ReportRange,
	Role,
	Satisfaction,
	ServiceCategory,
	ServiceCategoryInput,
	ServiceItem,
	ServiceItemInput,
	ServiceRequestInput,
	SavedView,
	SlaPolicy,
	Stats,
	TagCount,
	Ticket,
	TicketEvent,
	TicketFilters,
	UpdateTicketInput,
	UpdateUserInput,
	User
} from './types';

/** Error devuelto por la API, con los errores por campo cuando la validación falla (422). */
export class ApiError extends Error {
	constructor(
		readonly status: number,
		message: string,
		readonly fields: Record<string, string> = {},
		/** Cuerpo completo de la respuesta, por ejemplo la lista de empresas de un 409 al iniciar sesión. */
		readonly data: Record<string, unknown> | null = null
	) {
		super(message);
		this.name = 'ApiError';
	}
}

export interface ApiClientOptions {
	/** URL del backend sin la barra final. Vacía usa el mismo origen (el proxy de Vite en desarrollo). */
	baseUrl?: string;
	/** Devuelve el token actual, o null si no hay sesión. */
	getToken?: () => string | null;
	/** Se llama cuando la API responde 401, para cerrar la sesión. */
	onUnauthorized?: (message: string) => void;
	fetch?: typeof fetch;
}

type Query = Record<string, string | number | undefined | null>;

export function toQuery(params: Query): string {
	const search = new URLSearchParams();
	for (const [key, value] of Object.entries(params)) {
		if (value !== undefined && value !== null && value !== '') search.set(key, String(value));
	}
	const s = search.toString();
	return s ? `?${s}` : '';
}

export function createApiClient(options: ApiClientOptions = {}) {
	const baseUrl = (options.baseUrl ?? '') + '/api';
	const doFetch = options.fetch ?? ((...args: Parameters<typeof fetch>) => fetch(...args));

	/** Hace la petición y devuelve la respuesta si fue correcta; si no, lanza ApiError. */
	async function send(method: string, path: string, body?: unknown): Promise<Response> {
		const headers: Record<string, string> = {};
		const isForm = typeof FormData !== 'undefined' && body instanceof FormData;
		if (body !== undefined && !isForm) headers['Content-Type'] = 'application/json';
		const token = options.getToken?.();
		if (token) headers.Authorization = `Bearer ${token}`;

		let res: Response;
		try {
			res = await doFetch(baseUrl + path, {
				method,
				headers,
				body: body === undefined ? undefined : isForm ? (body as FormData) : JSON.stringify(body)
			});
		} catch {
			throw new ApiError(0, 'No se pudo conectar con el servidor');
		}

		if (!res.ok) {
			const data = await res.json().catch(() => null);
			const message = data?.error ?? `Error ${res.status}`;
			if (res.status === 401) options.onUnauthorized?.(message);
			throw new ApiError(res.status, message, data?.fields ?? {}, data);
		}
		return res;
	}

	async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
		const res = await send(method, path, body);
		return (res.status === 204 ? null : await res.json().catch(() => null)) as T;
	}

	async function download(path: string): Promise<Blob> {
		return (await send('GET', path)).blob();
	}

	return {
		health: () => request<{ status: string }>('GET', '/health'),

		/** Con varias empresas para ese email, responde 409 con `organizations` para elegir. */
		login: (email: string, password: string, org = '') =>
			request<AuthResponse>('POST', '/auth/login', { email, password, org }),
		/** org: slug del portal de la empresa; vacío = soporte de la plataforma. */
		register: (name: string, email: string, password: string, org = '') =>
			request<AuthResponse>('POST', '/auth/register', { name, email, password, org }),
		signup: (input: SignupInput) => request<AuthResponse>('POST', '/signup', input),
		getPortal: (slug: string) => request<OrgChoice>('GET', `/portal/${encodeURIComponent(slug)}`),
		getOrg: () => request<OrgUsage>('GET', '/org'),
		updateOrg: (name: string) => request<OrgUsage>('PATCH', '/org', { name }),
		requestUpgrade: (input: { agents: number; people: number; message: string }) =>
			request<null>('POST', '/org/upgrade', input),
		listPlatformOrgs: () => request<List<PlatformOrg>>('GET', '/platform/orgs'),
		updatePlatformOrg: (
			id: number,
			input: { people?: number; max_agents?: number; unlimited?: boolean; suspended?: boolean }
		) => request<Organization>('PATCH', `/platform/orgs/${id}`, input),
		me: () => request<User>('GET', '/me'),
		updateMe: (input: {
			name?: string;
			email?: string;
			current_password?: string;
			email_notifications?: boolean;
		}) => request<User>('PATCH', '/me', input),
		changePassword: (current_password: string, new_password: string) =>
			request<AuthResponse>('POST', '/me/password', { current_password, new_password }),
		forgotPassword: (email: string) => request<null>('POST', '/auth/forgot', { email }),
		resetPassword: (token: string, password: string) =>
			request<AuthResponse>('POST', '/auth/reset', { token, password }),

		listTickets: (filters: TicketFilters = {}) =>
			request<Page<Ticket>>('GET', '/tickets' + toQuery({ ...filters })),
		getTicket: (id: number) => request<Ticket>('GET', `/tickets/${id}`),
		createTicket: (input: CreateTicketInput) => request<Ticket>('POST', '/tickets', input),
		updateTicket: (id: number, input: UpdateTicketInput) =>
			request<Ticket>('PATCH', `/tickets/${id}`, input),
		bulkUpdate: (input: BulkUpdateInput) =>
			request<{ updated: number }>('POST', '/tickets/bulk', input),
		listEvents: (ticketId: number) =>
			request<List<TicketEvent>>('GET', `/tickets/${ticketId}/events`),
		rateTicket: (ticketId: number, rating: Exclude<Satisfaction, ''>, comment = '') =>
			request<Ticket>('POST', `/tickets/${ticketId}/satisfaction`, { rating, comment }),

		listAttachments: (ticketId: number) =>
			request<List<Attachment>>('GET', `/tickets/${ticketId}/attachments`),
		/** Sube un archivo a la descripción del ticket o, con commentId, a ese comentario. */
		uploadAttachment: (ticketId: number, file: File, commentId?: number) => {
			const form = new FormData();
			if (commentId) form.append('comment_id', String(commentId));
			form.append('file', file);
			return request<Attachment>('POST', `/tickets/${ticketId}/attachments`, form);
		},
		downloadAttachment: (id: number) => download(`/attachments/${id}`),
		deleteAttachment: (id: number) => request<null>('DELETE', `/attachments/${id}`),

		listComments: (ticketId: number) =>
			request<List<Comment>>('GET', `/tickets/${ticketId}/comments`),
		addComment: (ticketId: number, body: string, internal = false) =>
			request<Comment>('POST', `/tickets/${ticketId}/comments`, { body, internal }),

		listUsers: (role?: Role, options: { q?: string; all?: boolean } = {}) =>
			request<List<User>>(
				'GET',
				'/users' + toQuery({ role, q: options.q, active: options.all ? 'all' : undefined })
			),
		createUser: (input: CreateUserInput) => request<User>('POST', '/users', input),
		updateUser: (id: number, input: UpdateUserInput) =>
			request<User>('PATCH', `/users/${id}`, input),
		setUserRole: (id: number, role: Role) => request<User>('PATCH', `/users/${id}/role`, { role }),

		stats: () => request<Stats>('GET', '/stats'),

		/** Catálogo de servicios. Con all, un administrador ve también los servicios inactivos. */
		getCatalog: (all = false) =>
			request<{ categories: ServiceCategory[] }>(
				'GET',
				'/catalog' + toQuery({ all: all ? 1 : undefined })
			),
		requestService: (itemId: number, input: ServiceRequestInput) =>
			request<Ticket>('POST', `/catalog/items/${itemId}/request`, input),
		createServiceCategory: (input: ServiceCategoryInput) =>
			request<ServiceCategory>('POST', '/catalog/categories', input),
		updateServiceCategory: (id: number, input: ServiceCategoryInput) =>
			request<ServiceCategory>('PATCH', `/catalog/categories/${id}`, input),
		deleteServiceCategory: (id: number) => request<null>('DELETE', `/catalog/categories/${id}`),
		createServiceItem: (input: ServiceItemInput) =>
			request<ServiceItem>('POST', '/catalog/items', input),
		updateServiceItem: (id: number, input: ServiceItemInput) =>
			request<ServiceItem>('PATCH', `/catalog/items/${id}`, input),
		deleteServiceItem: (id: number) => request<null>('DELETE', `/catalog/items/${id}`),

		/** Activos (equipos). */
		listAssets: (filters: AssetFilters = {}) =>
			request<Page<Asset>>('GET', '/assets' + toQuery({ ...filters })),
		assetsSummary: () => request<AssetSummary>('GET', '/assets/summary'),
		myAssets: () => request<List<Asset>>('GET', '/assets/mine'),
		getAsset: (id: number) => request<AssetDetail>('GET', `/assets/${id}`),
		createAsset: (input: AssetInput) => request<Asset>('POST', '/assets', input),
		updateAsset: (id: number, input: AssetInput) => request<Asset>('PATCH', `/assets/${id}`, input),
		deleteAsset: (id: number) => request<null>('DELETE', `/assets/${id}`),
		listTicketAssets: (ticketId: number) =>
			request<List<Asset>>('GET', `/tickets/${ticketId}/assets`),
		linkTicketAsset: (ticketId: number, assetId: number) =>
			request<Asset>('POST', `/tickets/${ticketId}/assets`, { asset_id: assetId }),
		unlinkTicketAsset: (ticketId: number, assetId: number) =>
			request<null>('DELETE', `/tickets/${ticketId}/assets/${assetId}`),

		listTags: () => request<List<TagCount>>('GET', '/tags'),
		listCategories: () => request<List<Category>>('GET', '/categories'),
		createCategory: (name: string) => request<Category>('POST', '/categories', { name }),
		renameCategory: (id: number, name: string) =>
			request<Category>('PATCH', `/categories/${id}`, { name }),
		deleteCategory: (id: number) => request<null>('DELETE', `/categories/${id}`),

		listMacros: () => request<List<Macro>>('GET', '/macros'),
		createMacro: (input: MacroInput) => request<Macro>('POST', '/macros', input),
		updateMacro: (id: number, input: MacroInput) => request<Macro>('PATCH', `/macros/${id}`, input),
		deleteMacro: (id: number) => request<null>('DELETE', `/macros/${id}`),

		listSla: () => request<List<SlaPolicy>>('GET', '/sla'),
		updateSla: (items: SlaPolicy[]) => request<List<SlaPolicy>>('PUT', '/sla', { items }),

		listNotifications: (limit = 20) =>
			request<NotificationList>('GET', '/notifications' + toQuery({ limit })),
		readNotifications: (input: { ids?: number[]; ticket_id?: number } = {}) =>
			request<null>('POST', '/notifications/read', input),

		/** org: slug de la empresa; sin él, la del usuario (o la de la plataforma sin sesión). */
		listArticles: (q?: string, org?: string) =>
			request<List<Article>>('GET', '/articles' + toQuery({ q, org })),
		getArticle: (id: number) => request<Article>('GET', `/articles/${id}`),
		createArticle: (input: ArticleInput) => request<Article>('POST', '/articles', input),
		updateArticle: (id: number, input: ArticleInput) =>
			request<Article>('PATCH', `/articles/${id}`, input),
		deleteArticle: (id: number) => request<null>('DELETE', `/articles/${id}`),

		createLead: (input: LeadInput) => request<null>('POST', '/leads', input),
		listLeads: () => request<List<Lead>>('GET', '/leads'),
		setLeadHandled: (id: number, handled: boolean) =>
			request<Lead>('PATCH', `/leads/${id}`, { handled }),

		listViews: () => request<List<SavedView>>('GET', '/views'),
		createView: (name: string, query: string) =>
			request<SavedView>('POST', '/views', { name, query }),
		deleteView: (id: number) => request<null>('DELETE', `/views/${id}`),

		report: (range: ReportRange = {}) => request<Report>('GET', '/reports' + toQuery({ ...range })),
		exportTickets: (range: ReportRange = {}) =>
			download('/reports/tickets.csv' + toQuery({ ...range }))
	};
}

export type ApiClient = ReturnType<typeof createApiClient>;
