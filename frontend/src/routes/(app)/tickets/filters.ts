import {
	TICKET_KINDS,
	TICKET_PRIORITIES,
	TICKET_STATUSES,
	type TicketFilters
} from '#lib/api/types.js';

/** Lee los filtros de la URL descartando valores inválidos, para no enviar basura a la API. */
export function parseTicketFilters(params: Pick<URLSearchParams, 'get'>): TicketFilters {
	const filters: TicketFilters = {};

	const status = params.get('status');
	if (status && (TICKET_STATUSES as readonly string[]).includes(status)) {
		filters.status = status as TicketFilters['status'];
	}

	const priority = params.get('priority');
	if (priority && (TICKET_PRIORITIES as readonly string[]).includes(priority)) {
		filters.priority = priority as TicketFilters['priority'];
	}

	const assignee = params.get('assignee');
	if (assignee === 'me' || assignee === 'none') {
		filters.assignee = assignee;
	} else if (assignee && /^\d+$/.test(assignee)) {
		filters.assignee = Number(assignee);
	}

	const tag = params.get('tag')?.trim().toLowerCase();
	if (tag) filters.tag = tag;

	if (params.get('sla') === 'breached') filters.sla = 'breached';

	const kind = params.get('kind');
	if (kind && (TICKET_KINDS as readonly string[]).includes(kind)) {
		filters.kind = kind as TicketFilters['kind'];
	}

	const asset = params.get('asset');
	if (asset && /^\d+$/.test(asset)) filters.asset = Number(asset);

	const q = params.get('q')?.trim();
	if (q) filters.q = q;

	return filters;
}
