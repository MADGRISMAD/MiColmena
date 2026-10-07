<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import FileTextIcon from '@lucide/svelte/icons/file-text';
	import SearchIcon from '@lucide/svelte/icons/search';
	import { api, type Article, type OrgChoice } from '#lib/api/index.js';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Logo from '#lib/components/logo.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { isAuthenticated } from '#lib/stores/auth.js';

	// Portal público de una empresa: sus clientes abren tickets y leen su ayuda.
	const slug = $derived(page.params.slug ?? '');
	let org = $state<OrgChoice | null>(null);
	let missing = $state(false);
	let articles = $state<Article[] | null>(null);
	let query = $state('');

	$effect(() => {
		const s = slug;
		org = null;
		missing = false;
		api
			.getPortal(s)
			.then((o) => (org = o))
			.catch(() => (missing = true));
	});

	let requestId = 0;
	$effect(() => {
		const q = query.trim();
		const s = slug;
		const id = ++requestId;
		const timer = setTimeout(() => {
			api
				.listArticles(q || undefined, s)
				.then((r) => id === requestId && (articles = r.items))
				.catch(() => id === requestId && (articles = []));
		}, 250);
		return () => clearTimeout(timer);
	});

	function openTicket() {
		if ($isAuthenticated) goto(resolve('/(app)/tickets/new'));
		else goto(resolve(`/register?org=${encodeURIComponent(slug)}`));
	}
</script>

<svelte:head><title>{org ? `Soporte de ${org.name}` : 'Soporte'} · BeHIve</title></svelte:head>

{#if missing}
	<main class="grid min-h-screen place-items-center px-4">
		<EmptyState
			title="Este portal no existe"
			description="Revisa la dirección que te compartieron o escribe a la empresa."
		>
			<Button variant="outline" href={resolve('/')}>Ir a BeHIve</Button>
		</EmptyState>
	</main>
{:else}
	<div class="flex min-h-screen flex-col">
		<header class="border-b bg-background/85 backdrop-blur">
			<div class="mx-auto flex h-16 max-w-4xl items-center justify-between gap-4 px-4">
				<p class="flex min-w-0 items-center gap-3 font-semibold">
					<span class="hex size-7 shrink-0 bg-honey" aria-hidden="true"></span>
					<span class="truncate">{org?.name ?? ''}</span>
				</p>
				{#if $isAuthenticated}
					<Button size="sm" href={resolve('/(app)/tickets')}>Mis tickets</Button>
				{:else}
					<Button variant="ghost" size="sm" href={resolve(`/login?org=${encodeURIComponent(slug)}`)}
						>Iniciar sesión</Button
					>
				{/if}
			</div>
		</header>

		<main class="mx-auto w-full max-w-4xl flex-1 px-4 py-10">
			<section class="bg-honeycomb rounded-2xl border px-5 py-10 text-center sm:px-8">
				{#if org}
					<h1 class="text-3xl font-bold tracking-tight text-balance">Soporte de {org.name}</h1>
				{:else}
					<Skeleton class="mx-auto h-9 w-64" />
				{/if}
				<p class="mt-2 text-muted-foreground">
					Busca una respuesta o abre un ticket y el equipo te ayudará.
				</p>
				<div class="relative mx-auto mt-6 max-w-xl">
					<SearchIcon
						class="pointer-events-none absolute top-1/2 left-4 size-5 -translate-y-1/2 text-muted-foreground"
						aria-hidden="true"
					/>
					<input
						bind:value={query}
						type="search"
						aria-label="Buscar en la ayuda"
						placeholder="¿En qué podemos ayudarte?"
						class="h-12 w-full rounded-xl border bg-card pr-4 pl-12 shadow-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
					/>
				</div>
				<Button size="lg" class="mt-6" onclick={openTicket}>Abrir un ticket</Button>
			</section>

			<section class="mt-10" aria-labelledby="articles-title">
				<h2 id="articles-title" class="mb-4 font-semibold">Artículos de ayuda</h2>
				{#if articles === null}
					<div class="grid gap-2"><Skeleton class="h-16" /><Skeleton class="h-16" /></div>
				{:else if articles.length === 0}
					<p class="rounded-xl border border-dashed p-6 text-center text-sm text-muted-foreground">
						{query.trim() ? 'Nada coincide con tu búsqueda.' : 'Todavía no hay artículos.'}
						Abre un ticket y te respondemos.
					</p>
				{:else}
					<ul class="grid gap-2 sm:grid-cols-2">
						{#each articles as a (a.id)}
							<li>
								<a
									href={resolve('/help/[id]', { id: String(a.id) })}
									class="flex h-full gap-3 rounded-xl border bg-card p-4 shadow-xs transition-shadow hover:shadow-md"
								>
									<FileTextIcon class="mt-0.5 size-5 shrink-0 text-amber-600" aria-hidden="true" />
									<span class="min-w-0">
										<span class="block font-medium">{a.title}</span>
										<span class="mt-1 line-clamp-2 block text-sm text-muted-foreground"
											>{a.body}</span
										>
									</span>
								</a>
							</li>
						{/each}
					</ul>
				{/if}
			</section>
		</main>

		<footer class="border-t">
			<div
				class="mx-auto flex max-w-4xl flex-wrap items-center justify-between gap-3 px-4 py-6 text-xs text-muted-foreground"
			>
				<a href={resolve('/')} class="flex items-center gap-2">
					Funciona con <Logo class="origin-left scale-75" />
				</a>
				<a href={resolve('/privacidad')} class="hover:text-foreground">Aviso de privacidad</a>
			</div>
		</footer>
	</div>
{/if}
