<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import SearchIcon from '@lucide/svelte/icons/search';
	import {
		api,
		TICKET_PRIORITIES,
		TICKET_STATUSES,
		toQuery,
		type Ticket,
		type TicketFilters
	} from '#lib/api/index.js';
	import PageHeader from '#lib/components/page-header.svelte';
	import PriorityBadge from '#lib/components/priority-badge.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatRelative, priorityLabels, statusLabels } from '#lib/format.js';
	import { isStaff } from '#lib/stores/auth.js';
	import { parseTicketFilters } from './filters.js';

	let tickets = $state<Ticket[]>([]);
	let nextCursor = $state<number | null>(null);
	let loading = $state(true);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);

	// Los filtros viven en la URL: se pueden compartir y el botón Atrás funciona.
	const filters = $derived(parseTicketFilters(page.url.searchParams));
	// Texto del buscador: sigue a la URL, pero se puede editar antes de enviarlo.
	let search = $derived(filters.q ?? '');

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

	const hasFilters = $derived(Object.keys(filters).length > 0);
</script>

<PageHeader
	title={$isStaff ? 'Tickets' : 'Mis tickets'}
	description={$isStaff ? 'Todos los tickets de soporte.' : 'Las solicitudes que has abierto.'}
>
	{#snippet actions()}
		<Button href={resolve('/(app)/tickets/new')}>
			<PlusIcon aria-hidden="true" />
			Nuevo ticket
		</Button>
	{/snippet}
</PageHeader>

<div class="mb-4 flex flex-wrap gap-2">
	<form
		class="relative min-w-56 flex-1"
		role="search"
		onsubmit={(e) => {
			e.preventDefault();
			setFilter('q', search.trim());
		}}
	>
		<SearchIcon
			class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
			aria-hidden="true"
		/>
		<Input
			type="search"
			placeholder="Buscar por título o descripción…"
			aria-label="Buscar tickets"
			class="pl-8"
			bind:value={search}
		/>
	</form>

	<NativeSelect
		aria-label="Filtrar por estado"
		value={filters.status ?? ''}
		onchange={(e) => setFilter('status', e.currentTarget.value)}
	>
		<NativeSelectOption value="">Todos los estados</NativeSelectOption>
		{#each TICKET_STATUSES as status (status)}
			<NativeSelectOption value={status}>{statusLabels[status]}</NativeSelectOption>
		{/each}
	</NativeSelect>

	<NativeSelect
		aria-label="Filtrar por prioridad"
		value={filters.priority ?? ''}
		onchange={(e) => setFilter('priority', e.currentTarget.value)}
	>
		<NativeSelectOption value="">Todas las prioridades</NativeSelectOption>
		{#each TICKET_PRIORITIES as priority (priority)}
			<NativeSelectOption value={priority}>{priorityLabels[priority]}</NativeSelectOption>
		{/each}
	</NativeSelect>

	{#if $isStaff}
		<NativeSelect
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
		<Button variant="ghost" href={resolve('/(app)/tickets')}>Quitar filtros</Button>
	{/if}
</div>

{#if error}
	<Card.Root>
		<Card.Content class="flex items-center justify-between gap-4">
			<p class="text-muted-foreground">{error}</p>
			<Button variant="outline" onclick={() => load(filters)}>Reintentar</Button>
		</Card.Content>
	</Card.Root>
{:else if loading}
	<div class="grid gap-2" aria-busy="true">
		{#each { length: 5 }, i (i)}
			<Skeleton class="h-12" />
		{/each}
	</div>
{:else if tickets.length === 0}
	<Card.Root>
		<Card.Content class="py-10 text-center">
			<p class="font-medium">
				{hasFilters ? 'Ningún ticket coincide con los filtros.' : 'Todavía no hay tickets.'}
			</p>
			{#if !hasFilters}
				<Button class="mt-4" href={resolve('/(app)/tickets/new')}>Abrir el primero</Button>
			{/if}
		</Card.Content>
	</Card.Root>
{:else}
	<Card.Root class="py-0">
		<Table.Root>
			<Table.Header>
				<Table.Row>
					<Table.Head class="w-16">#</Table.Head>
					<Table.Head>Título</Table.Head>
					<Table.Head>Estado</Table.Head>
					<Table.Head>Prioridad</Table.Head>
					{#if $isStaff}
						<Table.Head class="hidden lg:table-cell">Solicitante</Table.Head>
						<Table.Head class="hidden md:table-cell">Asignado</Table.Head>
					{/if}
					<Table.Head class="hidden sm:table-cell">Actualizado</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each tickets as ticket (ticket.id)}
					<Table.Row class="relative">
						<Table.Cell class="text-muted-foreground tabular-nums">{ticket.id}</Table.Cell>
						<Table.Cell class="max-w-80 truncate font-medium">
							<a
								href={resolve('/(app)/tickets/[id]', { id: String(ticket.id) })}
								class="after:absolute after:inset-0 hover:underline"
							>
								{ticket.title}
							</a>
						</Table.Cell>
						<Table.Cell><StatusBadge status={ticket.status} /></Table.Cell>
						<Table.Cell><PriorityBadge priority={ticket.priority} /></Table.Cell>
						{#if $isStaff}
							<Table.Cell class="hidden lg:table-cell">{ticket.requester.name}</Table.Cell>
							<Table.Cell class="hidden text-muted-foreground md:table-cell">
								{ticket.assignee?.name ?? 'Sin asignar'}
							</Table.Cell>
						{/if}
						<Table.Cell class="hidden text-muted-foreground sm:table-cell">
							{formatRelative(ticket.updated_at)}
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
	</Card.Root>

	{#if nextCursor !== null}
		<div class="mt-4 flex justify-center">
			<Button variant="outline" disabled={loadingMore} onclick={() => load(filters, true)}>
				{loadingMore ? 'Cargando…' : 'Cargar más'}
			</Button>
		</div>
	{/if}
{/if}
