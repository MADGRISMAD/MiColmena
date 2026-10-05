// Backend simulado en memoria para las pruebas E2E. Intercepta /api/* en el navegador
// e imita las respuestas del backend en Go, así las pruebas no necesitan Go ni PostgreSQL.
import type { Page, Route } from '@playwright/test';
import type { Comment, Ticket, User } from '../src/lib/api/types';

const now = () => new Date().toISOString();

export const users = {
	agent: { id: 1, name: 'Luis Agente', email: 'luis@micolmena.dev', role: 'agent' },
	customer: { id: 2, name: 'Ana Cliente', email: 'ana@micolmena.dev', role: 'customer' }
} satisfies Record<string, Omit<User, 'created_at'>>;

const PASSWORD = 'secreto123';

export class FakeApi {
	tickets: Ticket[] = [];
	comments: Comment[] = [];
	requests: { method: string; path: string; body: unknown }[] = [];
	private nextId = 1;

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
			custom_fields: {},
			created_at: now(),
			updated_at: now(),
			resolved_at: null,
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

	private async handle(route: Route) {
		const req = route.request();
		const url = new URL(req.url());
		const path = url.pathname.replace(/^\/api/, '');
		const body = req.postData() ? JSON.parse(req.postData()!) : undefined;
		this.requests.push({ method: req.method(), path: path + url.search, body });

		const json = (status: number, data: unknown) => route.fulfill({ status, json: data });

		if (req.method() === 'POST' && path === '/auth/login') {
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

		const me = this.currentUser(route);
		if (!me) return json(401, { error: 'falta el token de acceso' });
		const staff = me.role !== 'customer';

		if (path === '/me') return json(200, me);

		if (path === '/stats') {
			return json(200, {
				by_status: { open: 2, in_progress: 1, waiting: 0, resolved: 0, closed: 0 },
				open_by_priority: { low: 0, medium: 1, high: 1, urgent: 1 },
				unassigned_open: 3,
				avg_resolution_hours_30d: 5.5
			});
		}

		if (path === '/users') {
			return json(200, { items: url.searchParams.get('role') === 'agent' ? [users.agent] : [] });
		}

		if (path === '/tickets' && req.method() === 'GET') {
			let items = this.tickets.filter((t) => staff || t.requester.id === me.id);
			const status = url.searchParams.get('status');
			if (status) items = items.filter((t) => t.status === status);
			const q = url.searchParams.get('q');
			if (q) items = items.filter((t) => t.title.toLowerCase().includes(q.toLowerCase()));
			return json(200, { items: [...items].reverse(), next_cursor: null });
		}

		if (path === '/tickets' && req.method() === 'POST') {
			return json(201, this.addTicket(body, me));
		}

		const match = path.match(/^\/tickets\/(\d+)(\/comments)?$/);
		const ticket = match && this.tickets.find((t) => t.id === Number(match[1]));
		if (!ticket || (!staff && ticket.requester.id !== me.id)) {
			return json(404, { error: 'ticket no encontrado' });
		}

		if (match![2]) {
			if (req.method() === 'POST') {
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

		if (req.method() === 'PATCH') {
			if ('assignee_id' in body) {
				ticket.assignee = body.assignee_id ? { id: users.agent.id, name: users.agent.name } : null;
				delete body.assignee_id;
			}
			Object.assign(ticket, body, { updated_at: now() });
		}
		return json(200, ticket);
	}
}
