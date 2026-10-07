// Backend simulado en memoria para las pruebas E2E. Intercepta /api/* en el navegador
// e imita las respuestas del backend en Go, así las pruebas no necesitan Go ni PostgreSQL.
import type { Page, Route } from '@playwright/test';
import type {
	Article,
	Attachment,
	Comment,
	Macro,
	Notification,
	SavedView,
	Ticket,
	TicketEvent,
	User
} from '../src/lib/api/types';

const now = () => new Date().toISOString();
const inHours = (h: number) => new Date(Date.now() + h * 3_600_000).toISOString();

const base = { active: true, email_notifications: true, org_id: 2 };
export const users = {
	agent: { id: 1, name: 'Luis Agente', email: 'luis@micolmena.dev', role: 'agent', ...base },
	customer: { id: 2, name: 'Ana Cliente', email: 'ana@micolmena.dev', role: 'customer', ...base },
	admin: { id: 4, name: 'Marta Admin', email: 'marta@micolmena.dev', role: 'admin', ...base },
	/** Administradora permanente de la plataforma (empresa 1). */
	owner: {
		id: 5,
		name: 'Dueña',
		email: 'madgrismad@gmail.com',
		role: 'admin',
		...base,
		org_id: 1,
		permanent: true
	}
} satisfies Record<string, Omit<User, 'created_at'>>;

const PASSWORD = 'secreto123';

export class FakeApi {
	tickets: Ticket[] = [];
	comments: Comment[] = [];
	events: TicketEvent[] = [];
	attachments: Attachment[] = [];
	notifications: Notification[] = [];
	macros: Macro[] = [];
	views: SavedView[] = [];
	articles: Article[] = [];
	categories = [{ id: 1, name: 'Facturación', created_at: now() }];
	org = {
		id: 2,
		name: 'Ferretería López',
		slug: 'ferreteria-lopez',
		people: 10,
		max_agents: 1 as number | null,
		suspended: false,
		created_at: now(),
		agents: 1,
		platform: false
	};
	signups: Record<string, unknown>[] = [];
	upgrades: Record<string, unknown>[] = [];
	/** Si la API debe rechazar la sesión con este mensaje (401). */
	revokeWith: string | null = null;
	leads: Record<string, string>[] = [];
	requests: { method: string; path: string; body: unknown }[] = [];
	private nextId = 1;
	private seq = 100;

	constructor() {
		this.addTicket({ title: 'No puedo iniciar sesión', priority: 'high' }, users.customer);
		this.addTicket({ title: 'Factura duplicada', priority: 'medium' }, users.customer);
		this.addTicket({ title: 'Error en la impresora', priority: 'urgent' }, { id: 3, name: 'Otro' });
	}

	addTicket(input: Partial<Ticket>, requester: { id: number; name: string }) {
		const ticket: Ticket = {
			id: this.nextId++,
			title: 'Ticket',
			description: '',
			status: 'open',
			priority: 'medium',
			category: '',
			requester: { id: requester.id, name: requester.name },
			assignee: null,
			tags: [],
			custom_fields: {},
			created_at: now(),
			updated_at: now(),
			resolved_at: null,
			first_response_at: null,
			first_response_due: inHours(4),
			resolution_due: inHours(24),
			satisfaction: '',
			satisfaction_comment: '',
			rated_at: null,
			...input
		};
		this.tickets.push(ticket);
		return ticket;
	}

	/** Deja la sesión iniciada como `role` antes de cargar la página. */
	async loginAs(page: Page, role: keyof typeof users) {
		await page.addInitScript((token) => localStorage.setItem('micolmena_token', token), role);
	}

	async install(page: Page) {
		await page.route('**/api/**', (route) => this.handle(route));
	}

	private currentUser(route: Route) {
		const token = route.request().headers()['authorization']?.replace('Bearer ', '');
		const user = users[token as keyof typeof users];
		return user ? { ...user, created_at: now() } : null;
	}

	private event(
		ticket: Ticket,
		actor: { id: number; name: string },
		kind: TicketEvent['kind'],
		from: string,
		to: string
	) {
		this.events.push({
			id: this.seq++,
			actor: { id: actor.id, name: actor.name },
			kind,
			old_value: from,
			new_value: to,
			created_at: now()
		} as TicketEvent & { ticket_id?: number });
		(this.events.at(-1) as TicketEvent & { ticket_id: number }).ticket_id = ticket.id;
	}

	private async handle(route: Route) {
		const req = route.request();
		const url = new URL(req.url());
		const path = url.pathname.replace(/^\/api/, '');
		const isJson = (req.headers()['content-type'] ?? '').includes('json');
		const body = req.postData() && isJson ? JSON.parse(req.postData()!) : undefined;
		this.requests.push({ method: req.method(), path: path + url.search, body });
		const method = req.method();

		const json = (status: number, data: unknown) => route.fulfill({ status, json: data });

		if (method === 'POST' && path === '/signup') {
			this.signups.push(body);
			return json(201, {
				token: 'admin',
				expires_at: now(),
				user: { ...users.admin, name: body.name, email: body.email, created_at: now() }
			});
		}
		const portal = path.match(/^\/portal\/([a-z0-9-]+)$/);
		if (portal) {
			return portal[1] === this.org.slug
				? json(200, { name: this.org.name, slug: this.org.slug })
				: json(404, { error: 'esa empresa no existe o no está disponible' });
		}
		// Un email con cuenta en dos empresas: hay que elegir.
		if (method === 'POST' && path === '/auth/login' && body.email === 'multi@correo.dev') {
			if (!body.org) {
				return json(409, {
					error: 'tienes cuenta en varias empresas: elige a cuál entrar',
					organizations: [
						{ slug: 'ferreteria-lopez', name: 'Ferretería López' },
						{ slug: 'taller-luna', name: 'Taller Luna' }
					]
				});
			}
			return json(200, {
				token: 'customer',
				expires_at: now(),
				user: { ...users.customer, created_at: now() }
			});
		}
		if (method === 'POST' && path === '/auth/login') {
			const user = Object.entries(users).find(([, u]) => u.email === body.email);
			if (!user || body.password !== PASSWORD) {
				return json(401, { error: 'email o contraseña incorrectos' });
			}
			return json(200, {
				token: user[0],
				expires_at: now(),
				user: { ...user[1], created_at: now() }
			});
		}
		if (method === 'POST' && path === '/auth/forgot') return route.fulfill({ status: 202 });
		if (method === 'POST' && path === '/leads') {
			this.leads.push(body);
			return route.fulfill({ status: 201 });
		}

		// La ayuda publicada es pública.
		if (path === '/articles' && method === 'GET') {
			const q = url.searchParams.get('q')?.toLowerCase();
			const items = this.articles.filter(
				(a) =>
					(a.published || this.currentUser(route)?.role !== 'customer') &&
					(!q || a.title.toLowerCase().includes(q))
			);
			return json(200, { items });
		}

		const articleMatch = path.match(/^\/articles\/(\d+)$/);
		if (articleMatch) {
			const article =
				method === 'GET' && this.articles.find((a) => a.id === Number(articleMatch[1]));
			return article ? json(200, article) : json(404, { error: 'artículo no encontrado' });
		}

		const me = this.currentUser(route);
		if (!me) return json(401, { error: 'falta el token de acceso' });
		if (this.revokeWith) return json(401, { error: this.revokeWith });

		if (path === '/org' && method === 'GET') {
			return json(200, me.org_id === 1 ? { ...this.org, id: 1, platform: true } : this.org);
		}
		if (path === '/org/upgrade') {
			this.upgrades.push(body);
			return route.fulfill({ status: 201 });
		}
		if (path === '/platform/orgs/2' && method === 'PATCH') {
			Object.assign(this.org, body);
			return json(200, { ...this.org, customers: 4, tickets: 12, last_ticket_at: now() });
		}
		if (path === '/platform/orgs') {
			return json(200, {
				items: [{ ...this.org, customers: 4, tickets: 12, last_ticket_at: now() }]
			});
		}
		const staff = me.role !== 'customer';

		if (path === '/stream') return route.fulfill({ status: 204 });
		if (path === '/me') return json(200, me);

		if (path === '/stats') {
			return json(200, {
				by_status: { open: 2, in_progress: 1, waiting: 0, resolved: 0, closed: 0 },
				open_by_priority: { low: 0, medium: 1, high: 1, urgent: 1 },
				unassigned_open: 3,
				avg_resolution_hours_30d: 5.5,
				sla_breached: 1,
				satisfaction_30d: { good: 3, bad: 1 }
			});
		}

		if (path === '/users') {
			const role = url.searchParams.get('role');
			const all = Object.values(users).map((u) => ({ ...u, created_at: now() }));
			return json(200, { items: role ? all.filter((u) => u.role === role) : all });
		}

		if (path === '/notifications') {
			return json(200, {
				items: this.notifications.filter(
					(n) => (n as Notification & { user_id: number }).user_id === me.id
				),
				unread: this.notifications.filter(
					(n) => (n as Notification & { user_id: number }).user_id === me.id && !n.read_at
				).length
			});
		}
		if (path === '/notifications/read') {
			for (const n of this.notifications) n.read_at ??= now();
			return route.fulfill({ status: 204 });
		}

		if (path === '/categories') return json(200, { items: this.categories });
		if (path === '/tags') {
			const counts: Record<string, number> = {};
			for (const t of this.tickets) for (const tag of t.tags) counts[tag] = (counts[tag] ?? 0) + 1;
			return json(200, { items: Object.entries(counts).map(([name, count]) => ({ name, count })) });
		}

		if (path === '/macros') {
			if (method === 'POST') {
				const macro = { id: this.seq++, ...body, created_at: now(), updated_at: now() };
				this.macros.push(macro);
				return json(201, macro);
			}
			return json(200, { items: this.macros });
		}

		if (path === '/views') {
			if (method === 'POST') {
				const view = { id: this.seq++, name: body.name, query: body.query, created_at: now() };
				this.views.push(view);
				return json(201, view);
			}
			return json(200, { items: this.views });
		}
		const viewMatch = path.match(/^\/views\/(\d+)$/);
		if (viewMatch && method === 'DELETE') {
			this.views = this.views.filter((v) => v.id !== Number(viewMatch[1]));
			return route.fulfill({ status: 204 });
		}

		if (path === '/articles' && method === 'POST') {
			const article = {
				id: this.seq++,
				...body,
				author: null,
				created_at: now(),
				updated_at: now()
			};
			this.articles.push(article);
			return json(201, article);
		}
		if (path === '/tickets' && method === 'GET') {
			let items = this.tickets.filter((t) => staff || t.requester.id === me.id);
			const status = url.searchParams.get('status');
			if (status) items = items.filter((t) => t.status === status);
			const tag = url.searchParams.get('tag');
			if (tag) items = items.filter((t) => t.tags.includes(tag));
			const q = url.searchParams.get('q');
			if (q) items = items.filter((t) => t.title.toLowerCase().includes(q.toLowerCase()));
			return json(200, { items: [...items].reverse(), next_cursor: null });
		}

		if (path === '/tickets' && method === 'POST') {
			return json(201, this.addTicket(body, me));
		}

		if (path === '/tickets/bulk' && method === 'POST') {
			for (const id of body.ids) {
				const t = this.tickets.find((x) => x.id === id);
				if (!t) continue;
				if (body.status) t.status = body.status;
				if (body.priority) t.priority = body.priority;
				if (body.add_tags) t.tags = [...new Set([...t.tags, ...body.add_tags])];
			}
			return json(200, { updated: body.ids.length });
		}

		const match = path.match(/^\/tickets\/(\d+)(\/[a-z]+)?$/);
		const ticket = match && this.tickets.find((t) => t.id === Number(match[1]));
		if (!ticket || (!staff && ticket.requester.id !== me.id)) {
			return json(404, { error: 'ticket no encontrado' });
		}
		const sub = match![2];

		if (sub === '/events') {
			return json(200, {
				items: this.events.filter(
					(e) => (e as TicketEvent & { ticket_id: number }).ticket_id === ticket.id
				)
			});
		}

		if (sub === '/attachments') {
			if (method === 'POST') {
				const raw = req.postData() ?? '';
				const filename = /filename="([^"]+)"/.exec(raw)?.[1] ?? 'archivo';
				const commentId = /name="comment_id"\r\n\r\n(\d+)/.exec(raw)?.[1];
				const a: Attachment = {
					id: this.seq++,
					ticket_id: ticket.id,
					comment_id: commentId ? Number(commentId) : null,
					uploader: { id: me.id, name: me.name },
					filename,
					content_type: 'text/plain',
					size: 12,
					internal: false,
					created_at: now()
				};
				this.attachments.push(a);
				return json(201, a);
			}
			return json(200, { items: this.attachments.filter((a) => a.ticket_id === ticket.id) });
		}

		if (sub === '/satisfaction') {
			Object.assign(ticket, {
				satisfaction: body.rating,
				satisfaction_comment: body.comment,
				rated_at: now()
			});
			return json(200, ticket);
		}

		if (sub === '/comments') {
			if (method === 'POST') {
				const comment: Comment = {
					id: this.comments.length + 1,
					ticket_id: ticket.id,
					author: { id: me.id, name: me.name },
					body: body.body,
					internal: body.internal,
					created_at: now()
				};
				this.comments.push(comment);
				return json(201, comment);
			}
			const items = this.comments.filter(
				(c) => c.ticket_id === ticket.id && (staff || !c.internal)
			);
			return json(200, { items });
		}

		if (method === 'PATCH') {
			if ('assignee_id' in body) {
				const before = ticket.assignee?.name ?? '';
				ticket.assignee = body.assignee_id ? { id: users.agent.id, name: users.agent.name } : null;
				this.event(ticket, me, 'assignee', before, ticket.assignee?.name ?? '');
				delete body.assignee_id;
			}
			if (body.status && body.status !== ticket.status) {
				this.event(ticket, me, 'status', ticket.status, body.status);
			}
			Object.assign(ticket, body, { updated_at: now() });
		}
		return json(200, ticket);
	}
}
