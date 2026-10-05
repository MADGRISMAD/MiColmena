import type {
	AuthResponse,
	Comment,
	CreateTicketInput,
	List,
	Page,
	Role,
	Stats,
	Ticket,
	TicketFilters,
	UpdateTicketInput,
	User
} from './types';

/** Error devuelto por la API, con los errores por campo cuando la validación falla (422). */
export class ApiError extends Error {
	constructor(
		readonly status: number,
		message: string,
		readonly fields: Record<string, string> = {}
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
	onUnauthorized?: () => void;
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

	async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
		const headers: Record<string, string> = {};
		if (body !== undefined) headers['Content-Type'] = 'application/json';
		const token = options.getToken?.();
		if (token) headers.Authorization = `Bearer ${token}`;

		let res: Response;
		try {
			res = await doFetch(baseUrl + path, {
				method,
				headers,
				body: body === undefined ? undefined : JSON.stringify(body)
			});
		} catch {
			throw new ApiError(0, 'No se pudo conectar con el servidor');
		}

		const data = res.status === 204 ? null : await res.json().catch(() => null);
		if (!res.ok) {
			if (res.status === 401) options.onUnauthorized?.();
			throw new ApiError(res.status, data?.error ?? `Error ${res.status}`, data?.fields ?? {});
		}
		return data as T;
	}

	return {
		health: () => request<{ status: string }>('GET', '/health'),

		login: (email: string, password: string) =>
			request<AuthResponse>('POST', '/auth/login', { email, password }),
		register: (name: string, email: string, password: string) =>
			request<AuthResponse>('POST', '/auth/register', { name, email, password }),
		me: () => request<User>('GET', '/me'),

		listTickets: (filters: TicketFilters = {}) =>
			request<Page<Ticket>>('GET', '/tickets' + toQuery({ ...filters })),
		getTicket: (id: number) => request<Ticket>('GET', `/tickets/${id}`),
		createTicket: (input: CreateTicketInput) => request<Ticket>('POST', '/tickets', input),
		updateTicket: (id: number, input: UpdateTicketInput) =>
			request<Ticket>('PATCH', `/tickets/${id}`, input),

		listComments: (ticketId: number) =>
			request<List<Comment>>('GET', `/tickets/${ticketId}/comments`),
		addComment: (ticketId: number, body: string, internal = false) =>
			request<Comment>('POST', `/tickets/${ticketId}/comments`, { body, internal }),

		listUsers: (role?: Role) => request<List<User>>('GET', '/users' + toQuery({ role })),
		setUserRole: (id: number, role: Role) => request<User>('PATCH', `/users/${id}/role`, { role }),

		stats: () => request<Stats>('GET', '/stats')
	};
}

export type ApiClient = ReturnType<typeof createApiClient>;
