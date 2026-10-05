<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { api, TICKET_PRIORITIES, TICKET_STATUSES, type Stats } from '#lib/api/index.js';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatHours, priorityLabels, statusLabels } from '#lib/format.js';
	import { isStaff } from '#lib/stores/auth.js';

	let stats = $state<Stats | null>(null);
	let error = $state<string | null>(null);

	async function load() {
		error = null;
		try {
			stats = await api.stats();
		} catch (err) {
			error = err instanceof Error ? err.message : 'No se pudieron cargar las estadísticas';
		}
	}

	$effect(() => {
		// El dashboard es para el equipo de soporte; los clientes van a sus tickets.
		if (!$isStaff) {
			goto(resolve('/(app)/tickets'), { replace: true });
			return;
		}
		load();
	});

	const openTotal = $derived(
		stats ? stats.by_status.open + stats.by_status.in_progress + stats.by_status.waiting : 0
	);
	const maxPriority = $derived(stats ? Math.max(1, ...Object.values(stats.open_by_priority)) : 1);
</script>

<PageHeader title="Dashboard" description="Estado actual del soporte." />

{#if error}
	<Card.Root>
		<Card.Content class="flex items-center justify-between gap-4">
			<p class="text-muted-foreground">{error}</p>
			<Button variant="outline" onclick={load}>Reintentar</Button>
		</Card.Content>
	</Card.Root>
{:else if !stats}
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		{#each { length: 4 }, i (i)}
			<Skeleton class="h-28" />
		{/each}
	</div>
{:else}
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		<a href={resolve('/(app)/tickets')} class="block">
			<Card.Root class="h-full transition-colors hover:bg-muted/50">
				<Card.Header>
					<Card.Description>Tickets sin resolver</Card.Description>
					<Card.Title class="text-3xl tabular-nums">{openTotal}</Card.Title>
				</Card.Header>
			</Card.Root>
		</a>
		<a href={resolve('/(app)/tickets?assignee=none')} class="block">
			<Card.Root class="h-full transition-colors hover:bg-muted/50">
				<Card.Header>
					<Card.Description>Sin asignar</Card.Description>
					<Card.Title class="text-3xl tabular-nums">{stats.unassigned_open}</Card.Title>
				</Card.Header>
			</Card.Root>
		</a>
		<a href={resolve('/(app)/tickets?priority=urgent')} class="block">
			<Card.Root class="h-full transition-colors hover:bg-muted/50">
				<Card.Header>
					<Card.Description>Urgentes abiertos</Card.Description>
					<Card.Title class="text-3xl tabular-nums">{stats.open_by_priority.urgent}</Card.Title>
				</Card.Header>
			</Card.Root>
		</a>
		<Card.Root>
			<Card.Header>
				<Card.Description>Tiempo medio de resolución (30 días)</Card.Description>
				<Card.Title class="text-3xl tabular-nums">
					{formatHours(stats.avg_resolution_hours_30d)}
				</Card.Title>
			</Card.Header>
		</Card.Root>
	</div>

	<div class="mt-4 grid gap-4 lg:grid-cols-2">
		<Card.Root>
			<Card.Header>
				<Card.Title>Tickets por estado</Card.Title>
			</Card.Header>
			<Card.Content>
				<dl class="grid gap-2">
					{#each TICKET_STATUSES as status (status)}
						<div class="flex items-center justify-between">
							<dt>
								<a href={resolve(`/(app)/tickets?status=${status}`)} class="hover:underline">
									{statusLabels[status]}
								</a>
							</dt>
							<dd class="font-medium tabular-nums">{stats.by_status[status]}</dd>
						</div>
					{/each}
				</dl>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header>
				<Card.Title>Sin resolver por prioridad</Card.Title>
			</Card.Header>
			<Card.Content>
				<dl class="grid gap-3">
					{#each [...TICKET_PRIORITIES].reverse() as priority (priority)}
						{@const count = stats.open_by_priority[priority]}
						<div class="grid grid-cols-[5rem_1fr_2.5rem] items-center gap-3">
							<dt class="text-sm">{priorityLabels[priority]}</dt>
							<div class="h-2 rounded-full bg-muted" aria-hidden="true">
								<div
									class="h-2 rounded-full bg-primary"
									style="width: {(count / maxPriority) * 100}%"
								></div>
							</div>
							<dd class="text-right font-medium tabular-nums">{count}</dd>
						</div>
					{/each}
				</dl>
			</Card.Content>
		</Card.Root>
	</div>
{/if}
