<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import ArrowRightIcon from '@lucide/svelte/icons/arrow-right';
	import ChevronsUpIcon from '@lucide/svelte/icons/chevrons-up';
	import InboxIcon from '@lucide/svelte/icons/inbox';
	import SirenIcon from '@lucide/svelte/icons/siren';
	import SmileIcon from '@lucide/svelte/icons/smile';
	import TimerIcon from '@lucide/svelte/icons/timer';
	import UserRoundXIcon from '@lucide/svelte/icons/user-round-x';
	import {
		api,
		TICKET_PRIORITIES,
		TICKET_STATUSES,
		type Stats,
		type Ticket
	} from '#lib/api/index.js';
	import PageHeader from '#lib/components/page-header.svelte';
	import PriorityBadge, { priorityStyles } from '#lib/components/priority-badge.svelte';
	import StatusBadge, { statusStyles } from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatHours, formatRelative, priorityLabels, statusLabels } from '#lib/format.js';
	import { liveEvent } from '#lib/live/index.js';
	import WelcomeChecklist from '#lib/components/welcome-checklist.svelte';
	import { org } from '#lib/stores/org.js';
	import { cn } from '#lib/utils.js';
	import { isStaff, user } from '#lib/stores/auth.js';

	let stats = $state<Stats | null>(null);
	let mine = $state<Ticket[] | null>(null);
	let unassigned = $state<Ticket[] | null>(null);
	let error = $state<string | null>(null);

	const pending = (t: Ticket) => t.status !== 'resolved' && t.status !== 'closed';

	async function load() {
		error = null;
		try {
			const [s, m, u] = await Promise.all([
				api.stats(),
				api.listTickets({ assignee: 'me', limit: 30 }),
				api.listTickets({ assignee: 'none', limit: 30 })
			]);
			stats = s;
			mine = m.items.filter(pending).slice(0, 5);
			unassigned = u.items.filter(pending).slice(0, 5);
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
		void $liveEvent.seq; // en tiempo real
		load();
	});

	const csat = $derived.by(() => {
		const r = stats?.satisfaction_30d;
		const n = r ? r.good + r.bad : 0;
		return r && n > 0 ? `${Math.round((r.good / n) * 100)} %` : '—';
	});

	const unresolved = $derived(
		stats ? stats.by_status.open + stats.by_status.in_progress + stats.by_status.waiting : 0
	);
	const total = $derived(stats ? Object.values(stats.by_status).reduce((a, b) => a + b, 0) : 0);
	const maxPriority = $derived(stats ? Math.max(1, ...Object.values(stats.open_by_priority)) : 1);

	const firstName = $derived($user?.name.split(' ')[0] ?? '');
	const today = new Intl.DateTimeFormat('es', {
		weekday: 'long',
		day: 'numeric',
		month: 'long'
	}).format(new Date());
</script>

<PageHeader
	eyebrow="Resumen"
	title={`Hola, ${firstName}`}
	description={`Así está el soporte hoy, ${today}.`}
/>

{#if $org && !$org.platform && $user?.role === 'admin'}
	<WelcomeChecklist org={$org} />
{/if}

{#snippet kpi(
	label: string,
	value: string | number,
	Icon: typeof InboxIcon,
	href: string | null,
	tone: string
)}
	<Card.Root
		class={cn('relative gap-0 overflow-hidden py-5', href && 'transition-shadow hover:shadow-md')}
	>
		<Card.Content class="flex-row items-start justify-between gap-3 px-5">
			<div>
				<p class="text-sm text-muted-foreground">{label}</p>
				<p class="mt-2 text-3xl font-semibold tracking-tight tabular-nums">{value}</p>
			</div>
			<span class={cn('hex grid size-11 shrink-0 place-items-center', tone)}>
				<Icon class="size-5" aria-hidden="true" />
			</span>
		</Card.Content>
		{#if href}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- href ya viene de resolve() -->
			<a {href} class="absolute inset-0" aria-label={`${label}: ver tickets`}></a>
		{/if}
	</Card.Root>
{/snippet}

{#if error}
	<Card.Root>
		<Card.Content class="flex items-center justify-between gap-4">
			<p class="text-muted-foreground">{error}</p>
			<Button variant="outline" onclick={load}>Reintentar</Button>
		</Card.Content>
	</Card.Root>
{:else if !stats}
	<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-busy="true">
		{#each { length: 4 }, i (i)}
			<Skeleton class="h-28 rounded-xl" />
		{/each}
	</div>
	<div class="mt-4 grid gap-4 lg:grid-cols-2">
		<Skeleton class="h-64 rounded-xl" />
		<Skeleton class="h-64 rounded-xl" />
	</div>
{:else}
	<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
		{@render kpi(
			'Tickets sin resolver',
			unresolved,
			InboxIcon,
			resolve('/(app)/tickets'),
			'bg-honey text-honey-foreground'
		)}
		{@render kpi(
			'Sin asignar',
			stats.unassigned_open,
			UserRoundXIcon,
			resolve('/(app)/tickets?assignee=none'),
			'bg-violet-100 text-violet-700 dark:bg-violet-500/20 dark:text-violet-300'
		)}
		{@render kpi(
			'Urgentes abiertos',
			stats.open_by_priority.urgent,
			ChevronsUpIcon,
			resolve('/(app)/tickets?priority=urgent'),
			'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-300'
		)}
		{@render kpi(
			'SLA vencido',
			stats.sla_breached,
			SirenIcon,
			resolve('/(app)/tickets?sla=breached'),
			stats.sla_breached > 0 ? 'bg-red-600 text-white' : 'bg-muted text-muted-foreground'
		)}
		{@render kpi(
			'Satisfacción (30 días)',
			csat,
			SmileIcon,
			null,
			'bg-sky-100 text-sky-700 dark:bg-sky-500/20 dark:text-sky-300'
		)}
		{@render kpi(
			'Resolución media (30 días)',
			formatHours(stats.avg_resolution_hours_30d),
			TimerIcon,
			null,
			'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300'
		)}
	</div>

	<div class="mt-4 grid gap-4 lg:grid-cols-2">
		<Card.Root>
			<Card.Header>
				<Card.Title>Tickets por estado</Card.Title>
				<Card.Description>{total} en total</Card.Description>
			</Card.Header>
			<Card.Content class="grid gap-5">
				<!-- Barra apilada: la proporción de cada estado de un vistazo. -->
				<div class="flex h-3 overflow-hidden rounded-full bg-muted" aria-hidden="true">
					{#each TICKET_STATUSES as status (status)}
						{#if stats.by_status[status] > 0}
							<div
								class={cn('h-full', statusStyles[status].dot)}
								style="width: {(stats.by_status[status] / Math.max(total, 1)) * 100}%"
							></div>
						{/if}
					{/each}
				</div>
				<dl class="grid gap-1">
					{#each TICKET_STATUSES as status (status)}
						<div class="flex items-center gap-3 rounded-md px-2 py-1.5 hover:bg-muted/60">
							<span class={cn('size-2.5 rounded-full', statusStyles[status].dot)} aria-hidden="true"
							></span>
							<dt class="flex-1">
								<a href={resolve(`/(app)/tickets?status=${status}`)} class="hover:underline">
									{statusLabels[status]}
								</a>
							</dt>
							<dd class="w-10 text-right text-sm text-muted-foreground tabular-nums">
								{total ? Math.round((stats.by_status[status] / total) * 100) : 0}%
							</dd>
							<dd class="w-8 text-right font-medium tabular-nums">{stats.by_status[status]}</dd>
						</div>
					{/each}
				</dl>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header>
				<Card.Title>Sin resolver por prioridad</Card.Title>
				<Card.Description>Lo urgente, primero</Card.Description>
			</Card.Header>
			<Card.Content>
				<dl class="grid gap-4">
					{#each [...TICKET_PRIORITIES].reverse() as priority (priority)}
						{@const count = stats.open_by_priority[priority]}
						<div class="grid grid-cols-[6.5rem_1fr_2rem] items-center gap-3">
							<dt>
								<a href={resolve(`/(app)/tickets?priority=${priority}`)} class="hover:underline">
									<PriorityBadge {priority} />
								</a>
							</dt>
							<div class="h-2.5 rounded-full bg-muted" aria-hidden="true">
								<div
									class={cn('h-2.5 rounded-full', priorityStyles[priority].bar)}
									style="width: {(count / maxPriority) * 100}%"
								></div>
							</div>
							<dd class="text-right font-medium tabular-nums">{count}</dd>
						</div>
					{/each}
				</dl>
				<p class="sr-only">
					{TICKET_PRIORITIES.map((p) => `${priorityLabels[p]}: ${stats?.open_by_priority[p]}`).join(
						', '
					)}
				</p>
			</Card.Content>
		</Card.Root>
	</div>

	<!-- Listas de trabajo -->
	<div class="mt-4 grid gap-4 lg:grid-cols-2">
		{#each [{ title: 'Asignados a mí', items: mine, href: resolve('/(app)/tickets?assignee=me'), empty: 'No tienes tickets pendientes. ¡Buen trabajo!' }, { title: 'Sin asignar', items: unassigned, href: resolve('/(app)/tickets?assignee=none'), empty: 'Todos los tickets tienen a alguien a cargo.' }] as list (list.title)}
			<Card.Root class="gap-0 pb-2">
				<Card.Header class="pb-3">
					<Card.Title>{list.title}</Card.Title>
					<Card.Action>
						<a
							href={list.href}
							class="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline dark:text-honey"
						>
							Ver todos <ArrowRightIcon class="size-3.5" aria-hidden="true" />
						</a>
					</Card.Action>
				</Card.Header>
				<Card.Content class="px-2">
					{#if !list.items?.length}
						<p class="px-4 py-6 text-center text-sm text-muted-foreground">{list.empty}</p>
					{:else}
						<ul>
							{#each list.items as ticket (ticket.id)}
								<li
									class="relative flex items-center gap-3 rounded-md px-4 py-2.5 hover:bg-muted/60"
								>
									<PriorityBadge priority={ticket.priority} compact />
									<a
										href={resolve('/(app)/tickets/[id]', { id: String(ticket.id) })}
										class="min-w-0 flex-1 truncate text-sm font-medium after:absolute after:inset-0"
									>
										{ticket.title}
									</a>
									<StatusBadge status={ticket.status} class="hidden sm:inline-flex" />
									<span class="w-20 text-right text-xs whitespace-nowrap text-muted-foreground">
										{formatRelative(ticket.updated_at)}
									</span>
								</li>
							{/each}
						</ul>
					{/if}
				</Card.Content>
			</Card.Root>
		{/each}
	</div>
{/if}
