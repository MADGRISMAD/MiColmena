// Backend simulado en memoria para las pruebas E2E. Intercepta /api/* en el navegador
// e imita las respuestas del backend en Go, así las pruebas no necesitan Go ni PostgreSQL.
import type { Page, Route } from '@playwright/test';
import type {
	Article,
	Asset,
	AssetEvent,
	Attachment,
	Comment,
	Macro,
	Notification,
	SavedView,
	ServiceCategory,
	ServiceItem,
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
	/** Catálogo de servicios (todas las categorías, también los servicios inactivos). */
	catalog: ServiceCategory[] = [
		{
			id: 1,
			name: 'Accesos y seguridad',
			description: 'Pide acceso a sistemas, carpetas o aplicaciones.',
			icon: 'key',
			position: 0,
			items: [
				{
					id: 1,
					category_id: 1,
					name: 'Acceso a un sistema',
					description: 'Pide acceso a una aplicación o carpeta compartida.',
					fields: [
						{ key: 'sistema', label: '¿A qué sistema o carpeta?', type: 'text', required: true },
						{
							key: 'nivel',
							label: 'Nivel de acceso',
							type: 'select',
							required: true,
							options: ['Solo lectura', 'Lectura y escritura']
						},
						{ key: 'motivo', label: 'Motivo', type: 'textarea', required: false }
					],
					priority: 'medium',
					ticket_category: '',
					active: true,
					position: 0,
					created_at: now()
				},
				{
					id: 2,
					category_id: 1,
					name: 'Restablecer contraseña',
					description: 'Recupera el acceso a tu cuenta.',
					fields: [{ key: 'cuenta', label: '¿De qué cuenta?', type: 'text', required: true }],
					priority: 'high',
					ticket_category: '',
					active: true,
					position: 1,
					created_at: now()
				}
			]
		},
		{
			id: 2,
			name: 'Equipo y dispositivos',
			description: 'Pide un equipo nuevo o un accesorio.',
			icon: 'laptop',
			position: 1,
			items: [
				{
					id: 3,
					category_id: 2,
					name: 'Equipo de cómputo nuevo',
					description: 'Laptop o computadora de escritorio.',
					fields: [],
					priority: 'medium',
					ticket_category: '',
					active: true,
					position: 0,
					created_at: now()
				}
			]
		}
	];
	/** Equipos (activos) y su actividad. */
	assets: Asset[] = [
		{
			id: 1,
			tag: 'LAP-001',
			name: 'Dell Latitude 5350',
			category: 'computer',
			model: 'Latitude 5350',
			serial: '4977X22',
			state: 'in_use',
			assigned_to: { id: 2, name: 'Ana Cliente' },
			location: 'Ventas',
			purchase_date: '2024-01-10',
			purchase_cost: 18500,
			warranty_until: '2099-01-01',
			notes: '',
			created_at: now(),
			updated_at: now(),
			open_tickets: 0
		},
		{
			id: 2,
			tag: 'PH-001',
			name: 'iPhone 15',
			category: 'phone',
			model: 'A3090',
			serial: 'F2LX9',
			state: 'in_stock',
			assigned_to: null,
			location: 'Almacén',
			purchase_date: '2023-03-01',
			purchase_cost: 17000,
			warranty_until: '2020-01-01',
			notes: '',
			created_at: now(),
			updated_at: now(),
			open_tickets: 0
		},
		{
			id: 3,
			tag: 'MON-001',
			name: 'Monitor Dell 24"',
			category: 'monitor',
			model: 'P2422H',
			serial: 'MON77',
			state: 'in_repair',
			assigned_to: null,
			location: 'Soporte',
			purchase_date: null,
			purchase_cost: null,
			warranty_until: null,
			notes: '',
			created_at: now(),
			updated_at: now(),
			open_tickets: 0
		}
	];
	assetEvents: (AssetEvent & { asset_id: number })[] = [
		{
			id: 1,
			asset_id: 1,
			actor: { id: 1, name: 'Luis Agente' },
			kind: 'created',
			old_value: '',
			new_value: '',
			created_at: now()
		}
	];
	/** Vínculos ticket–equipo. */
	ticketAssets: { ticket_id: number; asset_id: number }[] = [];
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
			kind: 'incident',
			catalog_item: null,
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
				mine_open: 2,
				requests_open: this.tickets.filter((t) => t.kind === 'request').length,
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

		if (path === '/catalog' && method === 'GET') {
			const all = url.searchParams.get('all') === '1' && me.role === 'admin';
			return json(200, {
				categories: this.catalog.map((c) => ({
					...c,
					items: c.items.filter((i) => all || i.active)
				}))
			});
		}
		const requestMatch = path.match(/^\/catalog\/items\/(\d+)\/request$/);
		if (requestMatch && method === 'POST') {
			const item = this.catalog
				.flatMap((c) => c.items)
				.find((i) => i.id === Number(requestMatch[1]));
			if (!item || !item.active) return json(404, { error: 'servicio no encontrado' });
			const fields: Record<string, string> = {};
			const lines: string[] = [];
			for (const f of item.fields) {
				const v = String(body.values?.[f.key] ?? '').trim();
				if (!v && f.required) fields[`values.${f.key}`] = 'es obligatorio';
				if (v) lines.push(`${f.label}: ${v}`);
			}
			if (Object.keys(fields).length) return json(422, { error: 'datos inválidos', fields });
			if (body.notes) lines.push('', body.notes);
			const ticket = this.addTicket(
				{
					title: item.name,
					description: lines.join('\n'),
					priority: item.priority,
					kind: 'request',
					catalog_item: { id: item.id, name: item.name },
					custom_fields: body.values
				},
				me
			);
			if (body.asset_id) this.ticketAssets.push({ ticket_id: ticket.id, asset_id: body.asset_id });
			return json(201, ticket);
		}
		if (path === '/catalog/categories' && method === 'POST') {
			if (!body.name)
				return json(422, { error: 'datos inválidos', fields: { name: 'es obligatorio' } });
			const c: ServiceCategory = {
				id: this.seq++,
				name: body.name,
				description: body.description ?? '',
				icon: body.icon ?? 'package',
				position: body.position ?? 0,
				items: []
			};
			this.catalog.push(c);
			return json(201, c);
		}
		const catMatch = path.match(/^\/catalog\/categories\/(\d+)$/);
		if (catMatch) {
			const idx = this.catalog.findIndex((c) => c.id === Number(catMatch[1]));
			if (idx < 0) return json(404, { error: 'categoría no encontrada' });
			if (method === 'DELETE') {
				this.catalog.splice(idx, 1);
				return route.fulfill({ status: 204 });
			}
			Object.assign(this.catalog[idx], {
				name: body.name,
				description: body.description ?? '',
				icon: body.icon ?? this.catalog[idx].icon
			});
			return json(200, this.catalog[idx]);
		}
		if (path === '/catalog/items' && method === 'POST') {
			const cat = this.catalog.find((c) => c.id === body.category_id);
			if (!cat)
				return json(422, { error: 'datos inválidos', fields: { category_id: 'no existe' } });
			if (!body.name)
				return json(422, { error: 'datos inválidos', fields: { name: 'es obligatorio' } });
			const item: ServiceItem = {
				id: this.seq++,
				category_id: cat.id,
				name: body.name,
				description: body.description ?? '',
				fields: (body.fields ?? []).map((f: ServiceItem['fields'][number], i: number) => ({
					...f,
					key: f.key || `campo_${i + 1}`
				})),
				priority: body.priority ?? 'medium',
				ticket_category: body.ticket_category ?? '',
				active: body.active ?? true,
				position: body.position ?? 0,
				created_at: now()
			};
			cat.items.push(item);
			return json(201, item);
		}
		const itemMatch = path.match(/^\/catalog\/items\/(\d+)$/);
		if (itemMatch) {
			const id = Number(itemMatch[1]);
			const cat = this.catalog.find((c) => c.items.some((i) => i.id === id));
			const item = cat?.items.find((i) => i.id === id);
			if (!cat || !item) return json(404, { error: 'servicio no encontrado' });
			if (method === 'DELETE') {
				cat.items = cat.items.filter((i) => i.id !== id);
				return route.fulfill({ status: 204 });
			}
			Object.assign(item, {
				...body,
				fields: (body.fields ?? []).map((f: ServiceItem['fields'][number], i: number) => ({
					...f,
					key: f.key || `campo_${i + 1}`
				}))
			});
			return json(200, item);
		}

		if (path.startsWith('/assets') || /^\/tickets\/\d+\/assets/.test(path)) {
			const res = this.handleAssets(route, method, path, url, body, me);
			if (res) return res;
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

	private openTicketCount(assetId: number) {
		return this.ticketAssets.filter(
			(l) =>
				l.asset_id === assetId &&
				!['resolved', 'closed'].includes(
					this.tickets.find((t) => t.id === l.ticket_id)?.status ?? ''
				)
		).length;
	}

	private withCounts(a: Asset): Asset {
		return { ...a, open_tickets: this.openTicketCount(a.id) };
	}

	private handleAssets(
		route: Route,
		method: string,
		path: string,
		url: URL,
		body: any, // eslint-disable-line @typescript-eslint/no-explicit-any
		me: { id: number; name: string; role: string }
	) {
		const json = (status: number, data: unknown) => route.fulfill({ status, json: data });
		const staff = me.role !== 'customer';
		const log = (asset_id: number, kind: AssetEvent['kind'], old_value = '', new_value = '') =>
			this.assetEvents.push({
				id: this.seq++,
				asset_id,
				actor: { id: me.id, name: me.name },
				kind,
				old_value,
				new_value,
				created_at: now()
			});
		const invalid = (fields: Record<string, string>) =>
			json(422, { error: 'datos inválidos', fields });

		if (path === '/assets/mine') {
			return json(200, {
				items: this.assets
					.filter((a) => a.assigned_to?.id === me.id && a.state !== 'retired')
					.map((a) => this.withCounts(a))
			});
		}
		if (!staff && path.startsWith('/assets')) return json(403, { error: 'no tienes permiso' });
		if (!staff) return null;

		if (path === '/assets/summary') {
			const live = this.assets.filter((a) => a.state !== 'retired');
			const today = new Date().toISOString().slice(0, 10);
			const soon = new Date(Date.now() + 60 * 86_400_000).toISOString().slice(0, 10);
			const count = (key: 'state' | 'category', v: string) =>
				this.assets.filter((a) => a[key] === v && (key === 'state' || a.state !== 'retired'))
					.length;
			return json(200, {
				total: this.assets.length,
				total_value: live.reduce((n, a) => n + (a.purchase_cost ?? 0), 0),
				by_state: Object.fromEntries(
					['in_stock', 'in_use', 'in_repair', 'retired'].map((k) => [k, count('state', k)])
				),
				by_category: Object.fromEntries(
					['computer', 'phone', 'monitor', 'network', 'peripheral', 'software', 'other'].map(
						(k) => [k, count('category', k)]
					)
				),
				warranty_expiring: live.filter(
					(a) => a.warranty_until && a.warranty_until >= today && a.warranty_until <= soon
				).length,
				warranty_expired: live.filter((a) => a.warranty_until && a.warranty_until < today).length,
				with_open_tickets: this.assets.filter((a) => this.openTicketCount(a.id) > 0).length,
				unassigned_in_use: this.assets.filter((a) => a.state === 'in_use' && !a.assigned_to).length
			});
		}
		if (path === '/assets' && method === 'GET') {
			const q = url.searchParams.get('q')?.toLowerCase();
			const state = url.searchParams.get('state');
			const category = url.searchParams.get('category');
			const assigned = url.searchParams.get('assigned');
			const warranty = url.searchParams.get('warranty');
			const today = new Date().toISOString().slice(0, 10);
			const soon = new Date(Date.now() + 60 * 86_400_000).toISOString().slice(0, 10);
			const items = this.assets
				.filter(
					(a) =>
						(!q ||
							[a.tag, a.name, a.model, a.serial, a.assigned_to?.name ?? '']
								.join(' ')
								.toLowerCase()
								.includes(q)) &&
						(!state || a.state === state) &&
						(!category || a.category === category) &&
						(!assigned ||
							(assigned === 'none'
								? !a.assigned_to
								: assigned === 'me'
									? a.assigned_to?.id === me.id
									: a.assigned_to?.id === Number(assigned))) &&
						(!warranty ||
							(a.state !== 'retired' &&
								!!a.warranty_until &&
								(warranty === 'expired'
									? a.warranty_until < today
									: a.warranty_until >= today && a.warranty_until <= soon)))
				)
				.map((a) => this.withCounts(a))
				.reverse();
			return json(200, { items, next_cursor: null });
		}
		if (path === '/assets' && method === 'POST') {
			const fields: Record<string, string> = {};
			if (!body.tag?.trim()) fields.tag = 'es obligatorio';
			if (!body.name?.trim()) fields.name = 'es obligatorio';
			if (this.assets.some((a) => a.tag.toLowerCase() === String(body.tag).toLowerCase()))
				fields.tag = 'ya existe un equipo con esa etiqueta';
			if (Object.keys(fields).length) return invalid(fields);
			const asset = this.buildAsset(this.seq++, body);
			this.assets.push(asset);
			log(asset.id, 'created');
			if (asset.assigned_to) log(asset.id, 'assigned', '', asset.assigned_to.name);
			return json(201, this.withCounts(asset));
		}
		const am = path.match(/^\/assets\/(\d+)$/);
		if (am) {
			const asset = this.assets.find((a) => a.id === Number(am[1]));
			if (!asset) return json(404, { error: 'equipo no encontrado' });
			if (method === 'DELETE') {
				if (me.role !== 'admin') return json(403, { error: 'no tienes permiso' });
				this.assets = this.assets.filter((a) => a !== asset);
				return route.fulfill({ status: 204 });
			}
			if (method === 'PATCH') {
				const next = this.buildAsset(asset.id, body, asset);
				if (next.state !== asset.state) log(asset.id, 'state', asset.state, next.state);
				if ((next.assigned_to?.id ?? 0) !== (asset.assigned_to?.id ?? 0))
					log(asset.id, 'assigned', asset.assigned_to?.name ?? '', next.assigned_to?.name ?? '');
				if (next.location !== asset.location)
					log(asset.id, 'location', asset.location, next.location);
				Object.assign(asset, next, { updated_at: now() });
				return json(200, this.withCounts(asset));
			}
			return json(200, {
				asset: this.withCounts(asset),
				events: this.assetEvents.filter((e) => e.asset_id === asset.id).reverse(),
				tickets: this.ticketAssets
					.filter((l) => l.asset_id === asset.id)
					.map((l) => this.tickets.find((t) => t.id === l.ticket_id)!)
					.filter(Boolean)
					.map((t) => ({
						id: t.id,
						title: t.title,
						status: t.status,
						priority: t.priority,
						kind: t.kind,
						created_at: t.created_at
					}))
			});
		}
		const tm = path.match(/^\/tickets\/(\d+)\/assets(?:\/(\d+))?$/);
		if (tm) {
			const ticketId = Number(tm[1]);
			if (method === 'GET') {
				return json(200, {
					items: this.ticketAssets
						.filter((l) => l.ticket_id === ticketId)
						.map((l) => this.assets.find((a) => a.id === l.asset_id)!)
						.filter(Boolean)
						.map((a) => this.withCounts(a))
				});
			}
			if (method === 'POST') {
				const asset = this.assets.find((a) => a.id === body.asset_id);
				if (!asset) return invalid({ asset_id: 'el equipo no existe' });
				if (!this.ticketAssets.some((l) => l.ticket_id === ticketId && l.asset_id === asset.id)) {
					this.ticketAssets.push({ ticket_id: ticketId, asset_id: asset.id });
					log(asset.id, 'ticket', '', `#${ticketId}`);
				}
				return json(200, this.withCounts(asset));
			}
			if (method === 'DELETE') {
				this.ticketAssets = this.ticketAssets.filter(
					(l) => !(l.ticket_id === ticketId && l.asset_id === Number(tm[2]))
				);
				return route.fulfill({ status: 204 });
			}
		}
		return null;
	}

	private buildAsset(
		id: number,
		body: any, // eslint-disable-line @typescript-eslint/no-explicit-any
		prev?: Asset
	): Asset {
		const assignee = body.assigned_to
			? Object.values(users).find((u) => u.id === body.assigned_to)
			: undefined;
		let state = body.state ?? 'in_stock';
		if (assignee && state === 'in_stock') state = 'in_use';
		if (!assignee && state === 'in_use') state = 'in_stock';
		return {
			id,
			tag: body.tag,
			name: body.name,
			category: body.category ?? 'computer',
			model: body.model ?? '',
			serial: body.serial ?? '',
			state,
			assigned_to: assignee ? { id: assignee.id, name: assignee.name } : null,
			location: body.location ?? '',
			purchase_date: body.purchase_date || null,
			purchase_cost: body.purchase_cost ?? null,
			warranty_until: body.warranty_until || null,
			notes: body.notes ?? '',
			created_at: prev?.created_at ?? now(),
			updated_at: now(),
			open_tickets: 0
		};
	}
}
