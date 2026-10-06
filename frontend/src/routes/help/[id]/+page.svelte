<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import { api, type Article } from '#lib/api/index.js';
	import EmptyState from '#lib/components/empty-state.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatDateTime } from '#lib/format.js';
	import { isAuthenticated, isStaff } from '#lib/stores/auth.js';

	let article = $state<Article | null>(null);
	let missing = $state(false);

	$effect(() => {
		const id = Number(page.params.id);
		void $isStaff;
		article = null;
		missing = false;
		api
			.getArticle(id)
			.then((a) => (article = a))
			.catch(() => (missing = true));
	});
</script>

<svelte:head><title>{article?.title ?? 'Ayuda'} · MiColmena</title></svelte:head>

{#if missing}
	<EmptyState title="Este artículo no existe o ya no está publicado">
		<Button variant="outline" href={resolve('/help')}>Volver a la ayuda</Button>
	</EmptyState>
{:else if !article}
	<Skeleton class="mb-4 h-8 w-2/3" />
	<Skeleton class="h-64" />
{:else}
	<nav class="mb-4 flex items-center gap-1 text-sm text-muted-foreground" aria-label="Ruta">
		<a href={resolve('/help')} class="hover:text-foreground">Ayuda</a>
		{#if article.category}
			<ChevronRightIcon class="size-3.5" aria-hidden="true" />
			<span>{article.category}</span>
		{/if}
	</nav>
	<article class="max-w-3xl">
		<div class="flex flex-wrap items-start justify-between gap-4">
			<h1 class="text-3xl font-bold tracking-tight text-balance">{article.title}</h1>
			{#if $isStaff}
				<Button
					variant="outline"
					size="sm"
					href={resolve('/help/[id]/edit', { id: String(article.id) })}
				>
					Editar
				</Button>
			{/if}
		</div>
		<p class="mt-2 text-sm text-muted-foreground">
			Actualizado el {formatDateTime(article.updated_at)}
			{#if !article.published}· <strong>Borrador</strong> (solo lo ve el equipo){/if}
		</p>
		<div class="mt-8 leading-relaxed whitespace-pre-wrap">{article.body}</div>
	</article>

	<aside class="mt-12 max-w-3xl rounded-xl border bg-card p-6">
		<p class="font-semibold">¿No resolvió tu duda?</p>
		<p class="mt-1 text-sm text-muted-foreground">Abre un ticket y el equipo te responderá.</p>
		<Button
			class="mt-4"
			href={$isAuthenticated ? resolve('/(app)/tickets/new') : resolve('/register')}
		>
			Abrir un ticket
		</Button>
	</aside>
{/if}
