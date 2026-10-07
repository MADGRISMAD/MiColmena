<script lang="ts">
	import { resolve } from '$app/paths';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import SearchIcon from '@lucide/svelte/icons/search';
	import { api, type ServiceCategory } from '#lib/api/index.js';
	import { catalogIcons } from '#lib/catalog-icons.js';
	import EmptyState from '#lib/components/empty-state.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { user } from '#lib/stores/auth.js';

	let categories = $state<ServiceCategory[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');

	const isAdmin = $derived($user?.role === 'admin');

	async function load() {
		loading = true;
		error = null;
		try {
			categories = (await api.getCatalog()).categories;
		} catch (err) {
			error = err instanceof Error ? err.message : 'No se pudo cargar el catálogo';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		load();
	});

	const visible = $derived.by(() => {
		const q = query.trim().toLowerCase();
		return categories
			.map((c) => ({
				...c,
				items: c.items.filter(
					(i) => !q || i.name.toLowerCase().includes(q) || i.description.toLowerCase().includes(q)
				)
			}))
			.filter((c) => c.items.length > 0);
	});
	const total = $derived(categories.reduce((n, c) => n + c.items.length, 0));
</script>

<PageHeader
	eyebrow="Servicios"
	title="Catálogo de servicios"
	description="Pide lo que necesitas y el equipo se encarga."
>
	{#snippet actions()}
		{#if isAdmin}
			<Button variant="outline" href={resolve('/(app)/admin/catalog')}>
				<PencilIcon aria-hidden="true" />
				Editar catálogo
			</Button>
		{/if}
	{/snippet}
</PageHeader>

{#if error}
	<Card.Root>
		<Card.Content class="flex items-center justify-between gap-4">
			<p class="text-muted-foreground">{error}</p>
			<Button variant="outline" onclick={load}>Reintentar</Button>
		</Card.Content>
	</Card.Root>
{:else if loading}
	<div class="grid gap-4 md:grid-cols-2" aria-busy="true">
		{#each { length: 4 }, i (i)}
			<Card.Root>
				<Card.Content class="grid gap-3">
					<Skeleton class="h-5 w-40" />
					<Skeleton class="h-4 w-full" />
					<Skeleton class="h-4 w-3/4" />
				</Card.Content>
			</Card.Root>
		{/each}
	</div>
{:else if total === 0}
	<Card.Root>
		<EmptyState
			title="Todavía no hay servicios"
			description={isAdmin
				? 'Crea categorías y servicios para que tu equipo pueda pedirlos desde aquí.'
				: 'Cuando haya servicios disponibles los verás aquí.'}
		>
			{#if isAdmin}
				<Button href={resolve('/(app)/admin/catalog')}>Editar catálogo</Button>
			{/if}
		</EmptyState>
	</Card.Root>
{:else}
	<div class="relative mb-5 max-w-md">
		<SearchIcon
			class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
			aria-hidden="true"
		/>
		<Input
			type="search"
			class="pl-9"
			placeholder="Buscar un servicio…"
			aria-label="Buscar un servicio"
			bind:value={query}
		/>
	</div>

	{#if visible.length === 0}
		<Card.Root>
			<EmptyState title="Ningún servicio coincide" description="Prueba con otras palabras.">
				<Button variant="outline" onclick={() => (query = '')}>Quitar búsqueda</Button>
			</EmptyState>
		</Card.Root>
	{:else}
		<div class="grid items-start gap-4 md:grid-cols-2">
			{#each visible as category (category.id)}
				{@const Icon = catalogIcons[category.icon] ?? catalogIcons.package}
				<Card.Root>
					<Card.Content class="grid gap-3">
						<div class="flex items-start gap-3">
							<span
								class="hex grid size-10 shrink-0 place-items-center bg-honey text-honey-foreground"
							>
								<Icon class="size-5" aria-hidden="true" />
							</span>
							<div class="min-w-0">
								<h2 class="font-semibold">{category.name}</h2>
								{#if category.description}
									<p class="text-sm text-muted-foreground">{category.description}</p>
								{/if}
							</div>
						</div>
						<ul class="grid gap-1">
							{#each category.items as item (item.id)}
								<li>
									<a
										href={resolve('/(app)/catalog/[id]', { id: String(item.id) })}
										class="block rounded-lg px-3 py-2 transition-colors hover:bg-honey/10"
									>
										<span class="font-medium">{item.name}</span>
										{#if item.description}
											<span class="block truncate text-sm text-muted-foreground"
												>{item.description}</span
											>
										{/if}
									</a>
								</li>
							{/each}
						</ul>
					</Card.Content>
				</Card.Root>
			{/each}
		</div>
	{/if}
{/if}
