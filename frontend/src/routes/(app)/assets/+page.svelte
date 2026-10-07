<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import LaptopIcon from '@lucide/svelte/icons/laptop';
	import PackageCheckIcon from '@lucide/svelte/icons/package-check';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import ShieldAlertIcon from '@lucide/svelte/icons/shield-alert';
	import ShieldXIcon from '@lucide/svelte/icons/shield-x';
	import TicketIcon from '@lucide/svelte/icons/ticket';
	import UserRoundXIcon from '@lucide/svelte/icons/user-round-x';
	import WarehouseIcon from '@lucide/svelte/icons/warehouse';
	import WrenchIcon from '@lucide/svelte/icons/wrench';
	import {
		api,
		ASSET_CATEGORIES,
		ASSET_STATES,
		toQuery,
		type Asset,
		type AssetFilters,
		type AssetSummary
	} from '#lib/api/index.js';
	import AssetStateBadge from '#lib/components/asset-state-badge.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import {
		assetCategoryLabels,
		assetStateLabels,
		formatDate,
		warrantyStatus
	} from '#lib/format.js';
	import { money } from '#lib/pricing.js';
	import { isStaff } from '#lib/stores/auth.js';
	import { cn } from '#lib/utils.js';

	let assets = $state<Asset[]>([]);
	let nextCursor = $state<number | null>(null);
	let loading = $state(true);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);
	let summary = $state<AssetSummary | null>(null);

	// Los filtros viven en la URL: se pueden compartir y el botón Atrás funciona.
	const filters = $derived.by<AssetFilters>(() => {
		const p = page.url.searchParams;
		const f: AssetFilters = {};
		const state = p.get('state');
		if (state && (ASSET_STATES as readonly string[]).includes(state)) {
			f.state = state as AssetFilters['state'];
		}
		const category = p.get('category');
		if (category && (ASSET_CATEGORIES as readonly string[]).includes(category)) {
			f.category = category as AssetFilters['category'];
		}
		const warranty = p.get('warranty');
		if (warranty === 'expiring' || warranty === 'expired') f.warranty = warranty;
		const q = p.get('q')?.trim();
		if (q) f.q = q;
		return f;
	});
	const hasFilters = $derived(Object.keys(filters).length > 0);

	let requestId = 0;
	async function load(f: AssetFilters, append = false) {
		const id = ++requestId;
		error = null;
		if (append) loadingMore = true;
		else loading = true;
		try {
			const res = await api.listAssets({
				...f,
				limit: 25,
				before: append ? (nextCursor ?? undefined) : undefined
			});
			if (id !== requestId) return;
			assets = append ? [...assets, ...res.items] : res.items;
			nextCursor = res.next_cursor;
		} catch (err) {
			if (id !== requestId) return;
			error = err instanceof Error ? err.message : 'No se pudieron cargar los equipos';
		} finally {
			if (id === requestId) {
				loading = false;
				loadingMore = false;
			}
		}
	}

	$effect(() => {
		if (!$isStaff) {
			goto(resolve('/(app)/tickets'), { replace: true });
			return;
		}
		load(filters);
	});

	$effect(() => {
		if (!$isStaff) return;
		api
			.assetsSummary()
			.then((s) => (summary = s))
			.catch(() => (summary = null));
	});

	function target(params: Record<string, string>) {
		const qs = toQuery(params);
		return qs ? resolve(`/(app)/assets${qs as `?${string}`}`) : resolve('/(app)/assets');
	}

	function setFilter(key: string, value: string) {
		goto(target({ ...Object.fromEntries(page.url.searchParams), [key]: value }), {
			replace: true,
			reset: false
		});
	}

	// Búsqueda con retardo; la URL manda y el campo solo la sigue cuando cambia desde fuera.
	let search = $state(page.url.searchParams.get('q') ?? '');
	let pushedQ = page.url.searchParams.get('q') ?? '';
	$effect(() => {
		const urlQ = filters.q ?? '';
		if (urlQ !== pushedQ) {
			pushedQ = urlQ;
			search = urlQ;
		}
	});
	let timer: ReturnType<typeof setTimeout>;
	function onSearch() {
		clearTimeout(timer);
		timer = setTimeout(() => {
			const q = search.trim();
			if (q === pushedQ) return;
			pushedQ = q;
			setFilter('q', q);
		}, 300);
	}

	// --- Tarjetas de resumen ---
	type Tile = {
		label: string;
		value: number;
		icon: typeof LaptopIcon;
		/** Parámetros del filtro; null = tarjeta informativa sin filtro. */
		params: Record<string, string> | null;
		band: string;
		tone: string;
	};
	const tiles = $derived<Tile[]>(
		summary
			? [
					{
						label: 'Total de equipos',
						value: summary.total,
						icon: LaptopIcon,
						params: {},
						band: 'border-t-amber-500',
						tone: 'bg-amber-100 text-amber-700 dark:bg-amber-400/15 dark:text-amber-300'
					},
					{
						label: 'En uso',
						value: summary.by_state.in_use,
						icon: PackageCheckIcon,
						params: { state: 'in_use' },
						band: 'border-t-emerald-500',
						tone: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-400/15 dark:text-emerald-300'
					},
					{
						label: 'En almacén',
						value: summary.by_state.in_stock,
						icon: WarehouseIcon,
						params: { state: 'in_stock' },
						band: 'border-t-sky-500',
						tone: 'bg-sky-100 text-sky-700 dark:bg-sky-400/15 dark:text-sky-300'
					},
					{
						label: 'En reparación',
						value: summary.by_state.in_repair,
						icon: WrenchIcon,
						params: { state: 'in_repair' },
						band: 'border-t-orange-500',
						tone: 'bg-orange-100 text-orange-700 dark:bg-orange-400/15 dark:text-orange-300'
					},
					{
						label: 'Garantía por vencer (60 días)',
						value: summary.warranty_expiring,
						icon: ShieldAlertIcon,
						params: { warranty: 'expiring' },
						band: 'border-t-yellow-500',
						tone: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-400/15 dark:text-yellow-300'
					},
					{
						label: 'Garantía vencida',
						value: summary.warranty_expired,
						icon: ShieldXIcon,
						params: { warranty: 'expired' },
						band: 'border-t-red-500',
						tone: 'bg-red-100 text-red-700 dark:bg-red-400/15 dark:text-red-300'
					},
					{
						label: 'Con tickets abiertos',
						value: summary.with_open_tickets,
						icon: TicketIcon,
						params: null,
						band: 'border-t-violet-500',
						tone: 'bg-violet-100 text-violet-700 dark:bg-violet-400/15 dark:text-violet-300'
					},
					{
						label: 'Sin responsable',
						value: summary.unassigned_in_use,
						icon: UserRoundXIcon,
						params: null,
						band: 'border-t-stone-400',
						tone: 'bg-stone-100 text-stone-700 dark:bg-stone-400/15 dark:text-stone-300'
					}
				]
			: []
	);

	/** Una tarjeta está activa si su filtro es exactamente el de la URL. */
	function tileActive(t: Tile) {
		if (!t.params) return false;
		const keys = Object.keys(t.params);
		const current = Object.keys(filters).filter((k) => k !== 'q');
		return (
			keys.length === current.length &&
			keys.every((k) => (filters as Record<string, string>)[k] === t.params![k])
		);
	}

	// --- Gráficas ---
	const categorySlices = $derived(
		summary
			? ASSET_CATEGORIES.map((c, i) => ({
					key: c,
					label: assetCategoryLabels[c],
					value: summary!.by_category[c] ?? 0,
					slot: i + 1
				}))
			: []
	);
	const categoryTotal = $derived(categorySlices.reduce((n, s) => n + s.value, 0));
	const RADIUS = 44;
	const CIRC = 2 * Math.PI * RADIUS;
	const arcs = $derived.by(() => {
		let offset = 0;
		return categorySlices
			.filter((s) => s.value > 0)
			.map((s) => {
				const len = (s.value / categoryTotal) * CIRC;
				// Un hueco de 2 px entre porciones; con una sola porción el anillo va completo.
				const gap = categorySlices.filter((x) => x.value > 0).length > 1 ? 2 : 0;
				const arc = { ...s, dash: `${Math.max(len - gap, 0.5)} ${CIRC}`, offset: -offset };
				offset += len;
				return arc;
			});
	});
	const stateRows = $derived(
		summary
			? ASSET_STATES.map((s) => ({
					key: s,
					label: assetStateLabels[s],
					value: summary!.by_state[s]
				}))
			: []
	);
	const maxState = $derived(Math.max(1, ...stateRows.map((r) => r.value)));
</script>

<PageHeader
	eyebrow="Inventario"
	title="Activos"
	description="Los equipos de la empresa: dónde están, quién los usa y qué tickets han tenido."
>
	{#snippet actions()}
		<Button href={resolve('/(app)/assets/new')}>
			<PlusIcon aria-hidden="true" />
			Nuevo equipo
		</Button>
	{/snippet}
</PageHeader>

{#if !summary}
	<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-busy="true">
		{#each { length: 8 }, i (i)}
			<Skeleton class="h-24 rounded-xl" />
		{/each}
	</div>
{:else}
	<ul class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-label="Resumen del inventario">
		{#each tiles as t (t.label)}
			{@const active = tileActive(t)}
			<li>
				<Card.Root
					class={cn(
						'relative h-full gap-0 overflow-hidden border-t-4 py-4',
						t.band,
						t.params && 'transition-shadow hover:shadow-md',
						active && 'ring-2 ring-primary'
					)}
				>
					<Card.Content class="flex-row items-start justify-between gap-3 px-5">
						<div>
							<p class="text-sm text-muted-foreground">{t.label}</p>
							<p class="mt-1.5 text-3xl font-semibold tracking-tight tabular-nums">{t.value}</p>
						</div>
						<span class={cn('hex grid size-10 shrink-0 place-items-center', t.tone)}>
							<t.icon class="size-5" aria-hidden="true" />
						</span>
					</Card.Content>
					{#if t.params}
						<a
							href={target(t.params)}
							class="absolute inset-0"
							aria-label={`${t.label}: ${t.value}. Filtrar la lista`}
							aria-current={active ? 'true' : undefined}
						></a>
					{/if}
				</Card.Root>
			</li>
		{/each}
	</ul>

	<div class="viz mt-4 grid gap-4 lg:grid-cols-2">
		<Card.Root class="gap-3 py-5">
			<Card.Header class="px-5">
				<Card.Title class="text-sm">Equipos por categoría</Card.Title>
				<Card.Description>Sin contar los retirados.</Card.Description>
			</Card.Header>
			<Card.Content class="flex flex-wrap items-center gap-6 px-5">
				{#if categoryTotal === 0}
					<p class="text-sm text-muted-foreground">Sin datos todavía.</p>
				{:else}
					<svg
						viewBox="0 0 120 120"
						class="size-40 shrink-0"
						role="img"
						aria-label={`Equipos por categoría: ${categorySlices
							.filter((s) => s.value > 0)
							.map((s) => `${s.label} ${s.value}`)
							.join(', ')}`}
					>
						<g transform="rotate(-90 60 60)">
							{#each arcs as a (a.key)}
								<circle
									cx="60"
									cy="60"
									r={RADIUS}
									fill="none"
									stroke-width="18"
									stroke-dasharray={a.dash}
									stroke-dashoffset={a.offset}
									style={`stroke: var(--series-${a.slot})`}
								>
									<title>{a.label}: {a.value}</title>
								</circle>
							{/each}
						</g>
						<text
							x="60"
							y="58"
							text-anchor="middle"
							class="fill-foreground text-[22px] font-semibold"
						>
							{categoryTotal}
						</text>
						<text x="60" y="73" text-anchor="middle" class="fill-muted-foreground text-[9px]">
							equipos
						</text>
					</svg>
					<ul class="grid min-w-44 flex-1 gap-1.5 text-sm" aria-label="Leyenda de categorías">
						{#each categorySlices.filter((s) => s.value > 0) as s (s.key)}
							<li>
								<a
									href={target({ category: s.key })}
									class="flex items-center gap-2 rounded-md px-1 py-0.5 hover:bg-muted"
								>
									<span
										class="size-3 shrink-0 rounded-[3px]"
										style={`background: var(--series-${s.slot})`}
										aria-hidden="true"
									></span>
									<span class="flex-1">{s.label}</span>
									<span class="font-medium tabular-nums">{s.value}</span>
									<span class="w-10 text-right text-xs text-muted-foreground tabular-nums">
										{Math.round((s.value / categoryTotal) * 100)} %
									</span>
								</a>
							</li>
						{/each}
					</ul>
				{/if}
			</Card.Content>
		</Card.Root>

		<Card.Root class="gap-3 py-5">
			<Card.Header class="px-5">
				<Card.Title class="text-sm">Equipos por estado</Card.Title>
				<Card.Description>
					Valor del inventario activo:
					<span class="font-medium text-foreground tabular-nums" data-testid="inventory-value">
						{money(summary.total_value)} MXN
					</span>
				</Card.Description>
			</Card.Header>
			<Card.Content class="px-5">
				<ul class="grid gap-3" aria-label="Equipos por estado">
					{#each stateRows as r (r.key)}
						<li>
							<a
								href={target({ state: r.key })}
								class="grid grid-cols-[6.5rem_1fr_2rem] items-center gap-3 rounded-md px-1 py-0.5 text-sm hover:bg-muted"
								title={`${r.label}: ${r.value}`}
							>
								<span>{r.label}</span>
								<span class="h-3 rounded-r-[4px] bg-muted/60" aria-hidden="true">
									<span
										class="block h-full rounded-r-[4px]"
										style={`width: ${(r.value / maxState) * 100}%; background: var(--series-1)`}
									></span>
								</span>
								<span class="text-right font-medium tabular-nums">{r.value}</span>
							</a>
						</li>
					{/each}
				</ul>
			</Card.Content>
		</Card.Root>
	</div>
{/if}

<!-- Filtros -->
<div class="mt-6 mb-4 flex flex-wrap items-center gap-2">
	<Input
		type="search"
		class="w-full sm:w-72"
		placeholder="Buscar por etiqueta, nombre, serie o responsable"
		aria-label="Buscar equipos"
		bind:value={search}
		oninput={onSearch}
	/>
	<NativeSelect
		size="sm"
		aria-label="Filtrar por estado"
		value={filters.state ?? ''}
		onchange={(e) => setFilter('state', e.currentTarget.value)}
	>
		<NativeSelectOption value="">Cualquier estado</NativeSelectOption>
		{#each ASSET_STATES as s (s)}
			<NativeSelectOption value={s}>{assetStateLabels[s]}</NativeSelectOption>
		{/each}
	</NativeSelect>
	<NativeSelect
		size="sm"
		aria-label="Filtrar por categoría"
		value={filters.category ?? ''}
		onchange={(e) => setFilter('category', e.currentTarget.value)}
	>
		<NativeSelectOption value="">Cualquier categoría</NativeSelectOption>
		{#each ASSET_CATEGORIES as c (c)}
			<NativeSelectOption value={c}>{assetCategoryLabels[c]}</NativeSelectOption>
		{/each}
	</NativeSelect>
	{#if filters.warranty}
		<span class="rounded-full bg-honey/20 px-3 py-0.5 text-sm font-medium">
			{filters.warranty === 'expiring' ? 'Garantía por vencer' : 'Garantía vencida'}
		</span>
	{/if}
	{#if hasFilters}
		<Button variant="ghost" size="sm" href={resolve('/(app)/assets')}>Quitar filtros</Button>
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
	<Card.Root class="gap-0 py-0" aria-busy="true">
		{#each { length: 5 }, i (i)}
			<div class="flex items-center gap-4 border-b px-4 py-4 last:border-0">
				<Skeleton class="h-4 w-16" />
				<Skeleton class="h-4 flex-1" />
				<Skeleton class="hidden h-5 w-20 rounded-full sm:block" />
			</div>
		{/each}
	</Card.Root>
{:else if assets.length === 0}
	<Card.Root>
		{#if hasFilters}
			<EmptyState
				title="Ningún equipo coincide con los filtros"
				description="Prueba con otra búsqueda o quita los filtros."
			>
				<Button variant="outline" href={resolve('/(app)/assets')}>Quitar filtros</Button>
			</EmptyState>
		{:else}
			<EmptyState
				title="Todavía no hay equipos"
				description="Da de alta laptops, teléfonos, monitores y demás para llevar su control."
			>
				<Button href={resolve('/(app)/assets/new')}>Agregar equipo</Button>
			</EmptyState>
		{/if}
	</Card.Root>
{:else}
	<Card.Root class="gap-0 overflow-hidden py-0">
		<Table.Root>
			<Table.Header>
				<Table.Row>
					<Table.Head>Etiqueta</Table.Head>
					<Table.Head>Equipo</Table.Head>
					<Table.Head>Categoría</Table.Head>
					<Table.Head>Estado</Table.Head>
					<Table.Head>Responsable</Table.Head>
					<Table.Head>Ubicación</Table.Head>
					<Table.Head>Garantía</Table.Head>
					<Table.Head class="text-right">Tickets abiertos</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each assets as a (a.id)}
					{@const warranty = warrantyStatus(a.warranty_until)}
					<Table.Row>
						<Table.Cell class="font-mono">
							<a
								href={resolve('/(app)/assets/[id]', { id: String(a.id) })}
								class="font-medium text-primary underline-offset-4 hover:underline dark:text-honey"
							>
								{a.tag}
							</a>
						</Table.Cell>
						<Table.Cell>
							<p class="font-medium">{a.name}</p>
							{#if a.model}<p class="text-xs text-muted-foreground">{a.model}</p>{/if}
						</Table.Cell>
						<Table.Cell>{assetCategoryLabels[a.category]}</Table.Cell>
						<Table.Cell><AssetStateBadge state={a.state} /></Table.Cell>
						<Table.Cell>
							{#if a.assigned_to}
								{a.assigned_to.name}
							{:else}
								<span class="text-muted-foreground">Sin asignar</span>
							{/if}
						</Table.Cell>
						<Table.Cell>{a.location || '—'}</Table.Cell>
						<Table.Cell>
							{#if warranty === 'expired'}
								<span
									class="rounded-full bg-red-50 px-2 py-0.5 text-xs font-medium text-red-800 ring-1 ring-red-600/20 ring-inset dark:bg-red-400/10 dark:text-red-300"
								>
									Vencida
								</span>
							{:else if warranty === 'expiring'}
								<span
									class="rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-800 ring-1 ring-amber-600/25 ring-inset dark:bg-amber-400/10 dark:text-amber-300"
								>
									Vence pronto
								</span>
							{:else}
								{formatDate(a.warranty_until)}
							{/if}
						</Table.Cell>
						<Table.Cell class="text-right tabular-nums">
							{#if a.open_tickets > 0}
								<a
									href={resolve(`/(app)/tickets?asset=${a.id}`)}
									class="font-medium text-primary underline-offset-4 hover:underline dark:text-honey"
									aria-label={`${a.open_tickets} tickets abiertos de ${a.tag}`}
								>
									{a.open_tickets}
								</a>
							{:else}
								<span class="text-muted-foreground">0</span>
							{/if}
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

<style>
	/* Paleta categórica de dataviz (slots 1-7), con pasos propios para modo oscuro. */
	.viz {
		--series-1: #2a78d6;
		--series-2: #eb6834;
		--series-3: #1baf7a;
		--series-4: #eda100;
		--series-5: #e87ba4;
		--series-6: #008300;
		--series-7: #4a3aa7;
	}
	@media (prefers-color-scheme: dark) {
		:global(:root:not([data-theme='light'])) .viz {
			--series-1: #3987e5;
			--series-2: #d95926;
			--series-3: #199e70;
			--series-4: #c98500;
			--series-5: #d55181;
			--series-6: #008300;
			--series-7: #9085e9;
		}
	}
	:global(:root[data-theme='dark']) .viz {
		--series-1: #3987e5;
		--series-2: #d95926;
		--series-3: #199e70;
		--series-4: #c98500;
		--series-5: #d55181;
		--series-6: #008300;
		--series-7: #9085e9;
	}
</style>
