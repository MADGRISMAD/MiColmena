<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import XIcon from '@lucide/svelte/icons/x';
	import {
		api,
		TICKET_PRIORITIES,
		toQuery,
		type Ticket,
		type TicketFilters,
		type TicketStatus
	} from '#lib/api/index.js';
	import EmptyState from '#lib/components/empty-state.svelte';
	import HexAvatar from '#lib/components/hex-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import PriorityBadge from '#lib/components/priority-badge.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatDateTime, formatRelative, priorityLabels } from '#lib/format.js';
	import { cn } from '#lib/utils.js';
	import { isStaff } from '#lib/stores/auth.js';
	import { parseTicketFilters } from './filters.js';

	let tickets = $state<Ticket[]>([]);
	let nextCursor = $state<number | null>(null);
	let loading = $state(true);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);

	// Los filtros viven en la URL: se pueden compartir y el botón Atrás funciona.
	const filters = $derived(parseTicketFilters(page.url.searchParams));

	let requestId = 0;
	async function load(f: TicketFilters, append = false) {
		const id = ++requestId;
		error = null;
		if (append) loadingMore = true;
		else loading = true;
		try {
			const res = await api.listTickets({
				...f,
				before: append ? (nextCursor ?? undefined) : undefined
			});
			if (id !== requestId) return; // Llegó una respuesta más nueva mientras tanto.
			tickets = append ? [...tickets, ...res.items] : res.items;
			nextCursor = res.next_cursor;
		} catch (err) {
			if (id !== requestId) return;
			error = err instanceof Error ? err.message : 'No se pudieron cargar los tickets';
		} finally {
			if (id === requestId) {
				loading = false;
				loadingMore = false;
			}
		}
	}

	$effect(() => {
		load(filters);
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
			: filters.assignee === 'me'
				? 'Asignados a mí'
				: filters.assignee === 'none'
					? 'Sin asignar'
					: 'Todos los tickets'
	);
	const description = $derived(
		$isStaff ? 'Solicitudes de soporte de todos los clientes.' : 'Las solicitudes que has abierto.'
	);

	const hasFilters = $derived(Object.keys(filters).length > 0);
</script>

<PageHeader eyebrow={$isStaff ? 'Vista' : undefined} {title} {description} />

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
			</NativeSelect>
		{/if}

		{#if hasFilters}
			<Button variant="ghost" size="sm" href={resolve('/(app)/tickets')}>Quitar filtros</Button>
		{/if}
	</div>
</div>

{#if filters.q}
	<div class="mb-4 flex items-center gap-2 text-sm">
		<span class="text-muted-foreground">Resultados para</span>
		<span
			class="inline-flex items-center gap-1 rounded-full bg-honey/20 py-0.5 pr-1 pl-3 font-medium"
		>
			“{filters.q}”
			<button
				type="button"
				class="rounded-full p-0.5 hover:bg-honey/30"
				onclick={() => setFilter('q', '')}
				aria-label="Quitar búsqueda"
			>
				<XIcon class="size-3.5" aria-hidden="true" />
			</button>
		</span>
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
		{#if hasFilters}
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
					<tr class="relative border-b transition-colors last:border-0 hover:bg-honey/5">
						<td class="px-4 py-3 font-mono text-xs text-muted-foreground tabular-nums">
							#{ticket.id}
						</td>
						<td class="max-w-0 px-4 py-3">
							<a
								href={resolve('/(app)/tickets/[id]', { id: String(ticket.id) })}
								class="block truncate font-medium after:absolute after:inset-0 hover:text-primary dark:hover:text-honey"
							>
								{ticket.title}
							</a>
							{#if ticket.category}
								<span class="mt-0.5 block truncate text-xs text-muted-foreground">
									{ticket.category}
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
