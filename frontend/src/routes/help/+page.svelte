<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import FileTextIcon from '@lucide/svelte/icons/file-text';
	import SearchIcon from '@lucide/svelte/icons/search';
	import { api, type Article } from '#lib/api/index.js';
	import EmptyState from '#lib/components/empty-state.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatRelative } from '#lib/format.js';
	import { isAuthenticated, isStaff } from '#lib/stores/auth.js';

	let articles = $state<Article[] | null>(null);
	let query = $state(page.url.searchParams.get('q') ?? '');

	$effect(() => {
		const q = page.url.searchParams.get('q') ?? '';
		void $isStaff; // los agentes ven también los borradores
		articles = null;
		api
			.listArticles(q || undefined)
			.then((r) => (articles = r.items))
			.catch(() => (articles = []));
	});

	function search(event: SubmitEvent) {
		event.preventDefault();
		const q = query.trim();
		goto(q ? resolve(`/help?q=${encodeURIComponent(q)}`) : resolve('/help'), { replace: true });
	}

	const grouped = $derived.by(() => {
		const groups: Record<string, Article[]> = {};
		for (const a of articles ?? []) (groups[a.category || 'General'] ??= []).push(a);
		return Object.entries(groups);
	});
	const searching = $derived(!!page.url.searchParams.get('q'));
</script>

<svelte:head><title>Centro de ayuda · MiColmena</title></svelte:head>

<section class="bg-honeycomb mb-10 rounded-2xl border px-5 py-10 text-center sm:px-6 sm:py-12">
	<h1 class="text-3xl font-bold tracking-tight">¿En qué podemos ayudarte?</h1>
	<p class="mt-2 text-muted-foreground">Busca en las guías del equipo de soporte.</p>
	<form role="search" class="relative mx-auto mt-6 max-w-xl" onsubmit={search}>
		<SearchIcon
			class="pointer-events-none absolute top-1/2 left-4 size-5 -translate-y-1/2 text-muted-foreground"
			aria-hidden="true"
		/>
		<input
			bind:value={query}
			type="search"
			aria-label="Buscar en la ayuda"
			placeholder="Por ejemplo: impresora, factura, contraseña…"
			class="h-12 w-full rounded-xl border bg-card pr-4 pl-12 shadow-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
		/>
	</form>
</section>

{#if articles === null}
	<div class="grid gap-3"><Skeleton class="h-16" /><Skeleton class="h-16" /></div>
{:else if articles.length === 0}
	<EmptyState
		title={searching ? 'No encontramos artículos sobre eso' : 'Todavía no hay artículos'}
		description="Si no encuentras la respuesta, abre un ticket y el equipo te ayudará."
	>
		<Button href={$isAuthenticated ? resolve('/(app)/tickets/new') : resolve('/register')}>
			Abrir un ticket
		</Button>
	</EmptyState>
{:else}
	<div class="grid gap-8">
		{#each grouped as [category, items] (category)}
			<section>
				<h2 class="mb-3 text-sm font-semibold tracking-wide text-muted-foreground uppercase">
					{category}
				</h2>
				<ul class="grid gap-2 sm:grid-cols-2">
					{#each items as a (a.id)}
						<li>
							<a
								href={resolve('/help/[id]', { id: String(a.id) })}
								class="flex h-full gap-3 rounded-xl border bg-card p-4 shadow-xs transition-shadow hover:shadow-md"
							>
								<FileTextIcon class="mt-0.5 size-5 shrink-0 text-amber-600" aria-hidden="true" />
								<span class="min-w-0">
									<span class="block font-medium">
										{a.title}
										{#if !a.published}
											<span class="ml-1 rounded bg-muted px-1.5 text-xs font-normal">Borrador</span>
										{/if}
									</span>
									<span class="mt-1 line-clamp-2 block text-sm text-muted-foreground">{a.body}</span
									>
									<span class="mt-1 block text-xs text-muted-foreground">
										Actualizado {formatRelative(a.updated_at)}
									</span>
								</span>
							</a>
						</li>
					{/each}
				</ul>
			</section>
		{/each}
	</div>
{/if}
