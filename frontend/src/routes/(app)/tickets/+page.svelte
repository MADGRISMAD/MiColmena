<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import BookmarkPlusIcon from '@lucide/svelte/icons/bookmark-plus';
	import XIcon from '@lucide/svelte/icons/x';
	import { toast } from 'svelte-sonner';
	import {
		api,
		TICKET_PRIORITIES,
		TICKET_STATUSES,
		toQuery,
		type BulkUpdateInput,
		type Ticket,
		type TicketFilters,
		type TicketStatus,
		type User
	} from '#lib/api/index.js';
	import EmptyState from '#lib/components/empty-state.svelte';
	import HexAvatar from '#lib/components/hex-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import PriorityBadge from '#lib/components/priority-badge.svelte';
	import SlaBadge from '#lib/components/sla-badge.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import TagChip from '#lib/components/tag-chip.svelte';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatDateTime, formatRelative, priorityLabels, statusLabels } from '#lib/format.js';
	import { onLive } from '#lib/live/index.js';
	import { cn } from '#lib/utils.js';
	import { isStaff } from '#lib/stores/auth.js';
	import { saveView } from '#lib/stores/views.js';
	import { parseTicketFilters } from './filters.js';

	let tickets = $state<Ticket[]>([]);
	let nextCursor = $state<number | null>(null);
	let loading = $state(true);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);
	let agents = $state<User[]>([]);
	let selected = $state<number[]>([]);
	let bulkBusy = $state(false);

	// Los filtros viven en la URL: se pueden compartir y el botón Atrás funciona.
	const filters = $derived(parseTicketFilters(page.url.searchParams));

	let requestId = 0;
	async function load(f: TicketFilters, append = false, quiet = false) {
		const id = ++requestId;
		error = null;
		if (append) loadingMore = true;
		else if (!quiet) loading = true;
		try {
			const res = await api.listTickets({
				...f,
				before: append ? (nextCursor ?? undefined) : undefined
			});
			if (id !== requestId) return; // Llegó una respuesta más nueva mientras tanto.
			tickets = append ? [...tickets, ...res.items] : res.items;
			nextCursor = res.next_cursor;
			if (!append) selected = selected.filter((sid) => res.items.some((t) => t.id === sid));
		} catch (err) {
			if (id !== requestId) return;
			if (!quiet) error = err instanceof Error ? err.message : 'No se pudieron cargar los tickets';
		} finally {
			if (id === requestId) {
				loading = false;
				loadingMore = false;
			}
		}
	}

	$effect(() => {
		selected = [];
		load(filters);
	});

	// En tiempo real: si cambia un ticket, se recarga la lista sin parpadeos.
	$effect(() =>
		onLive((event) => {
			if (event.type === 'notification') return;
			if (!loadingMore && nextCursor === null) load(filters, false, true);
		})
	);

	$effect(() => {
		if (!$isStaff) return;
		Promise.all([api.listUsers('admin'), api.listUsers('agent')])
			.then(([a, b]) => (agents = [...a.items, ...b.items]))
			.catch(() => (agents = []));
	});

	function setFilter(key: keyof TicketFilters, value: string) {
		const qs = toQuery({ ...Object.fromEntries(page.url.searchParams), [key]: value });
		const target = qs ? resolve(`/(app)/tickets${qs as `?${string}`}`) : resolve('/(app)/tickets');
		goto(target, { replace: true, reset: false });
	}

	const statusTabs: { value: TicketStatus | ''; label: string }[] = [
		{ value: '', label: 'Todos' },
		{ value: 'open', label: 'Abiertos' },
		{ value: 'in_progress', label: 'En curso' },
		{ value: 'waiting', label: 'En espera' },
		{ value: 'resolved', label: 'Resueltos' },
		{ value: 'closed', label: 'Cerrados' }
	];

	const title = $derived(
		!$isStaff
			? 'Mis tickets'
			: filters.sla === 'breached'
				? 'SLA vencido'
				: filters.kind === 'request'
					? 'Solicitudes'
					: filters.asset
						? `Tickets del equipo #${filters.asset}`
						: filters.assignee === 'me'
							? 'Asignados a mí'
							: filters.assignee === 'none'
								? 'Sin asignar'
								: 'Todos los tickets'
	);
	const description = $derived(
		filters.sla === 'breached'
			? 'Tickets pendientes que superaron su plazo de respuesta o resolución.'
			: $isStaff
				? 'Solicitudes de soporte de todos los clientes.'
				: 'Las solicitudes que has abierto.'
	);

	const hasFilters = $derived(Object.keys(filters).length > 0);

	async function saveCurrentView() {
		const name = prompt('Nombre de la vista', title)?.trim();
		if (!name) return;
		try {
			await saveView(name, page.url.search);
			toast.success('Vista guardada en la barra lateral');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo guardar la vista');
		}
	}

	// --- Selección y acciones masivas ---
	const allSelected = $derived(tickets.length > 0 && selected.length === tickets.length);

	function toggle(id: number) {
		selected = selected.includes(id) ? selected.filter((s) => s !== id) : [...selected, id];
	}

	function toggleAll() {
		selected = allSelected ? [] : tickets.map((t) => t.id);
	}

	async function bulk(changes: Omit<BulkUpdateInput, 'ids'>, message: string) {
		bulkBusy = true;
		try {
			const res = await api.bulkUpdate({ ids: selected, ...changes });
			toast.success(`${message}: ${res.updated} ${res.updated === 1 ? 'ticket' : 'tickets'}`);
			selected = [];
			await load(filters, false, true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo aplicar el cambio');
		} finally {
			bulkBusy = false;
		}
	}

	function bulkTag() {
		const tag = prompt('Etiqueta para añadir')?.trim();
		if (tag) bulk({ add_tags: [tag] }, 'Etiqueta añadida');
	}
</script>

<PageHeader eyebrow={$isStaff ? 'Vista' : undefined} {title} {description}>
	{#snippet actions()}
		{#if hasFilters}
			<Button variant="outline" size="sm" onclick={saveCurrentView}>
				<BookmarkPlusIcon aria-hidden="true" />
				Guardar vista
			</Button>
		{/if}
	{/snippet}
</PageHeader>

<!-- Pestañas de estado y filtros. -->
<div class="mb-4 flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
	<div class="-mx-4 overflow-x-auto px-4 xl:mx-0 xl:px-0">
		<div
			class="inline-flex gap-1 rounded-lg bg-muted p-1"
			role="group"
			aria-label="Filtrar por estado"
		>
			{#each statusTabs as tab (tab.value)}
				{@const active = (filters.status ?? '') === tab.value}
				<button
					type="button"
					aria-pressed={active}
					onclick={() => setFilter('status', tab.value)}
					class={cn(
						'rounded-md px-3 py-1.5 text-sm font-medium whitespace-nowrap text-muted-foreground transition-colors hover:text-foreground',
						active && 'bg-card text-foreground shadow-sm'
					)}
				>
					{tab.label}
				</button>
			{/each}
		</div>
	</div>

	<div class="flex flex-wrap items-center gap-2">
		<NativeSelect
			size="sm"
			aria-label="Filtrar por prioridad"
			value={filters.priority ?? ''}
			onchange={(e) => setFilter('priority', e.currentTarget.value)}
		>
			<NativeSelectOption value="">Cualquier prioridad</NativeSelectOption>
			{#each [...TICKET_PRIORITIES].reverse() as priority (priority)}
				<NativeSelectOption value={priority}>{priorityLabels[priority]}</NativeSelectOption>
			{/each}
		</NativeSelect>

		{#if $isStaff}
			<NativeSelect
				size="sm"
				aria-label="Filtrar por asignación"
				value={filters.assignee === undefined ? '' : String(filters.assignee)}
				onchange={(e) => setFilter('assignee', e.currentTarget.value)}
			>
				<NativeSelectOption value="">Cualquier agente</NativeSelectOption>
				<NativeSelectOption value="me">Asignados a mí</NativeSelectOption>
				<NativeSelectOption value="none">Sin asignar</NativeSelectOption>
				{#each agents as agent (agent.id)}
					<NativeSelectOption value={String(agent.id)}>{agent.name}</NativeSelectOption>
				{/each}
			</NativeSelect>
		{/if}

		{#if hasFilters}
			<Button variant="ghost" size="sm" href={resolve('/(app)/tickets')}>Quitar filtros</Button>
		{/if}
	</div>
</div>

{#if filters.q || filters.tag}
	<div class="mb-4 flex flex-wrap items-center gap-2 text-sm">
		<span class="text-muted-foreground">Filtrando por</span>
		{#each [{ key: 'q' as const, label: filters.q ? `“${filters.q}”` : '' }, { key: 'tag' as const, label: filters.tag ? `#${filters.tag}` : '' }] as chip (chip.key)}
			{#if chip.label}
				<span
					class="inline-flex items-center gap-1 rounded-full bg-honey/20 py-0.5 pr-1 pl-3 font-medium"
				>
					{chip.label}
					<button
						type="button"
						class="rounded-full p-0.5 hover:bg-honey/30"
						onclick={() => setFilter(chip.key, '')}
						aria-label={chip.key === 'q' ? 'Quitar búsqueda' : 'Quitar etiqueta'}
					>
						<XIcon class="size-3.5" aria-hidden="true" />
					</button>
				</span>
			{/if}
		{/each}
	</div>
{/if}

<!-- Barra de acciones masivas -->
{#if $isStaff && selected.length > 0}
	<div
		class="sticky top-16 z-10 mb-3 flex flex-wrap items-center gap-2 rounded-xl border border-honey/50 bg-card p-2 pl-4 shadow-md"
		role="toolbar"
		aria-label="Acciones para los tickets seleccionados"
	>
		<span class="mr-2 text-sm font-medium">
			{selected.length}
			{selected.length === 1 ? 'seleccionado' : 'seleccionados'}
		</span>
		<NativeSelect
			size="sm"
			aria-label="Cambiar estado"
			value=""
			disabled={bulkBusy}
			onchange={(e) => {
				const status = e.currentTarget.value as TicketStatus;
				if (status) bulk({ status }, `Estado: ${statusLabels[status]}`);
				e.currentTarget.value = '';
			}}
		>
			<NativeSelectOption value="">Estado…</NativeSelectOption>
			{#each TICKET_STATUSES as status (status)}
				<NativeSelectOption value={status}>{statusLabels[status]}</NativeSelectOption>
			{/each}
		</NativeSelect>
		<NativeSelect
			size="sm"
			aria-label="Cambiar prioridad"
			value=""
			disabled={bulkBusy}
			onchange={(e) => {
				const priority = e.currentTarget.value as Ticket['priority'];
				if (priority) bulk({ priority }, `Prioridad: ${priorityLabels[priority]}`);
				e.currentTarget.value = '';
			}}
		>
			<NativeSelectOption value="">Prioridad…</NativeSelectOption>
			{#each [...TICKET_PRIORITIES].reverse() as priority (priority)}
				<NativeSelectOption value={priority}>{priorityLabels[priority]}</NativeSelectOption>
			{/each}
		</NativeSelect>
		<NativeSelect
			size="sm"
			aria-label="Asignar a"
			value=""
			disabled={bulkBusy}
			onchange={(e) => {
				const value = e.currentTarget.value;
				if (value === 'none') bulk({ assignee_id: null }, 'Sin asignar');
				else if (value) bulk({ assignee_id: Number(value) }, 'Asignados');
				e.currentTarget.value = '';
			}}
		>
			<NativeSelectOption value="">Asignar a…</NativeSelectOption>
			<NativeSelectOption value="none">Nadie</NativeSelectOption>
			{#each agents as agent (agent.id)}
				<NativeSelectOption value={String(agent.id)}>{agent.name}</NativeSelectOption>
			{/each}
		</NativeSelect>
		<Button variant="outline" size="sm" disabled={bulkBusy} onclick={bulkTag}>Etiquetar…</Button>
		<Button variant="ghost" size="sm" class="ml-auto" onclick={() => (selected = [])}>
			Quitar selección
		</Button>
	</div>
{/if}

{#if error}
	<Card.Root>
		<Card.Content class="flex items-center justify-between gap-4">
			<p class="text-muted-foreground">{error}</p>
			<Button variant="outline" onclick={() => load(filters)}>Reintentar</Button>
		</Card.Content>
	</Card.Root>
{:else if loading}
	<Card.Root class="gap-0 py-0" aria-busy="true">
		{#each { length: 6 }, i (i)}
			<div class="flex items-center gap-4 border-b px-4 py-4 last:border-0">
				<Skeleton class="h-4 w-10" />
				<Skeleton class="h-4 flex-1" />
				<Skeleton class="hidden h-5 w-20 rounded-full sm:block" />
				<Skeleton class="hex hidden size-6 md:block" />
			</div>
		{/each}
	</Card.Root>
{:else if tickets.length === 0}
	<Card.Root>
		{#if filters.sla === 'breached' && Object.keys(filters).length === 1}
			<EmptyState
				title="Nada vencido"
				description="Todos los tickets pendientes están dentro de su plazo."
			/>
		{:else if hasFilters}
			<EmptyState
				title="Ningún ticket coincide con los filtros"
				description="Prueba con otro estado o quita la búsqueda."
			>
				<Button variant="outline" href={resolve('/(app)/tickets')}>Quitar filtros</Button>
			</EmptyState>
		{:else}
			<EmptyState
				title="Todavía no hay tickets"
				description={$isStaff
					? 'Cuando un cliente abra una solicitud, aparecerá aquí.'
					: 'Cuéntanos qué necesitas y el equipo de soporte te responderá.'}
			>
				<Button href={resolve('/(app)/tickets/new')}>Abrir el primero</Button>
			</EmptyState>
		{/if}
	</Card.Root>
{:else}
	<!-- Escritorio: tabla. -->
	<Card.Root class="hidden gap-0 overflow-hidden py-0 md:flex">
		<table class="w-full text-sm">
			<thead class="border-b bg-muted/50 text-left text-xs whitespace-nowrap text-muted-foreground">
				<tr>
					{#if $isStaff}
						<th class="w-10 py-2.5 pl-4">
							<input
								type="checkbox"
								class="size-4 accent-amber-500"
								checked={allSelected}
								indeterminate={selected.length > 0 && !allSelected}
								onchange={toggleAll}
								aria-label="Seleccionar todos"
							/>
						</th>
					{/if}
					<th class="w-20 px-4 py-2.5 font-medium">ID</th>
					<th class="w-full px-4 py-2.5 font-medium">Asunto</th>
					<th class="px-4 py-2.5 font-medium">Estado</th>
					<th class="px-4 py-2.5 font-medium">Prioridad</th>
					{#if $isStaff}
						<th class="hidden px-4 py-2.5 font-medium xl:table-cell">Solicitante</th>
						<th class="px-4 py-2.5 font-medium">Asignado</th>
					{/if}
					<th class="hidden px-4 py-2.5 text-right font-medium lg:table-cell">Actualizado</th>
				</tr>
			</thead>
			<tbody>
				{#each tickets as ticket (ticket.id)}
					{@const checked = selected.includes(ticket.id)}
					<tr
						class={cn(
							'relative border-b transition-colors last:border-0 hover:bg-honey/5',
							checked && 'bg-honey/10'
						)}
					>
						{#if $isStaff}
							<td class="relative z-10 py-3 pl-4">
								<input
									type="checkbox"
									class="size-4 accent-amber-500"
									{checked}
									onchange={() => toggle(ticket.id)}
									aria-label={`Seleccionar #${ticket.id}`}
								/>
							</td>
						{/if}
						<td class="px-4 py-3 font-mono text-xs text-muted-foreground tabular-nums">
							#{ticket.id}
						</td>
						<td class="max-w-0 px-4 py-3">
							<span class="flex items-center gap-2">
								<a
									href={resolve('/(app)/tickets/[id]', { id: String(ticket.id) })}
									class="block min-w-0 truncate font-medium after:absolute after:inset-0 hover:text-primary dark:hover:text-honey"
								>
									{ticket.title}
								</a>
								{#if ticket.kind === 'request'}
									<Badge variant="outline" class="relative z-10" title={ticket.catalog_item?.name}
										>Solicitud</Badge
									>
								{/if}
							</span>
							{#if ticket.category || ticket.tags.length > 0 || $isStaff}
								<span class="mt-1 flex items-center gap-1.5 overflow-hidden text-xs">
									{#if $isStaff}<SlaBadge {ticket} />{/if}
									{#if ticket.category}
										<span class="truncate text-muted-foreground">{ticket.category}</span>
									{/if}
									{#each ticket.tags.slice(0, 3) as tag (tag)}
										<TagChip {tag} />
									{/each}
								</span>
							{/if}
						</td>
						<td class="px-4 py-3"><StatusBadge status={ticket.status} /></td>
						<td class="px-4 py-3 whitespace-nowrap"><PriorityBadge priority={ticket.priority} /></td
						>
						{#if $isStaff}
							<td class="hidden px-4 py-3 xl:table-cell">
								<span class="flex items-center gap-2">
									<HexAvatar name={ticket.requester.name} id={ticket.requester.id} size="xs" />
									<span class="truncate">{ticket.requester.name}</span>
								</span>
							</td>
							<td class="px-4 py-3">
								<span class="flex items-center gap-2">
									<HexAvatar name={ticket.assignee?.name} id={ticket.assignee?.id} size="xs" />
									<span class={cn('truncate', !ticket.assignee && 'text-muted-foreground')}>
										{ticket.assignee?.name ?? 'Sin asignar'}
									</span>
								</span>
							</td>
						{/if}
						<td
							class="hidden px-4 py-3 text-right whitespace-nowrap text-muted-foreground lg:table-cell"
							title={formatDateTime(ticket.updated_at)}
						>
							{formatRelative(ticket.updated_at)}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</Card.Root>

	<!-- Móvil: tarjetas, para que nada quede cortado. -->
	<ul class="grid gap-2 md:hidden">
		{#each tickets as ticket (ticket.id)}
			<li class="relative rounded-xl border bg-card p-4 shadow-xs">
				<div class="mb-1.5 flex items-center justify-between gap-2">
					<span class="font-mono text-xs text-muted-foreground">#{ticket.id}</span>
					<StatusBadge status={ticket.status} />
				</div>
				<a
					href={resolve('/(app)/tickets/[id]', { id: String(ticket.id) })}
					class="block font-medium after:absolute after:inset-0"
				>
					{ticket.title}
				</a>
				{#if ticket.kind === 'request'}
					<Badge variant="outline" class="relative z-10 mt-1" title={ticket.catalog_item?.name}
						>Solicitud</Badge
					>
				{/if}
				{#if $isStaff || ticket.tags.length > 0}
					<div class="mt-2 flex flex-wrap gap-1.5">
						{#if $isStaff}<SlaBadge {ticket} />{/if}
						{#each ticket.tags.slice(0, 3) as tag (tag)}
							<TagChip {tag} />
						{/each}
					</div>
				{/if}
				<div class="mt-3 flex items-center gap-3 text-xs text-muted-foreground">
					<PriorityBadge priority={ticket.priority} class="text-xs" />
					{#if $isStaff}
						<span class="flex min-w-0 items-center gap-1.5">
							<HexAvatar name={ticket.assignee?.name} id={ticket.assignee?.id} size="xs" />
							<span class="truncate">{ticket.assignee?.name ?? 'Sin asignar'}</span>
						</span>
					{/if}
					<span class="ml-auto whitespace-nowrap">{formatRelative(ticket.updated_at)}</span>
				</div>
			</li>
		{/each}
	</ul>

	{#if nextCursor !== null}
		<div class="mt-4 flex justify-center">
			<Button variant="outline" disabled={loadingMore} onclick={() => load(filters, true)}>
				{loadingMore ? 'Cargando…' : 'Cargar más'}
			</Button>
		</div>
	{/if}
{/if}
