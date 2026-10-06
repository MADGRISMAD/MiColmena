<script lang="ts">
	import MessageSquareQuoteIcon from '@lucide/svelte/icons/message-square-quote';
	import { resolve } from '$app/paths';
	import { api, type Macro } from '#lib/api/index.js';

	let { onpick }: { onpick: (macro: Macro) => void } = $props();

	let open = $state(false);
	let macros = $state<Macro[] | null>(null);
	let search = $state('');
	let root = $state<HTMLDivElement | null>(null);

	async function toggle() {
		open = !open;
		if (open && macros === null) {
			macros = await api
				.listMacros()
				.then((r) => r.items)
				.catch(() => []);
		}
	}

	const filtered = $derived(
		(macros ?? []).filter((m) =>
			(m.title + ' ' + m.body).toLowerCase().includes(search.trim().toLowerCase())
		)
	);
</script>

<svelte:window
	onclick={(e) => {
		if (open && root && !root.contains(e.target as Node)) open = false;
	}}
/>

<div class="relative" bind:this={root}>
	<button
		type="button"
		class="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-muted-foreground hover:bg-accent hover:text-foreground"
		aria-expanded={open}
		onclick={toggle}
	>
		<MessageSquareQuoteIcon class="size-3.5" aria-hidden="true" />
		Respuestas guardadas
	</button>
	{#if open}
		<div
			class="absolute bottom-full left-0 z-30 mb-2 w-80 overflow-hidden rounded-xl border bg-popover text-popover-foreground shadow-xl"
		>
			<div class="border-b p-2">
				<!-- svelte-ignore a11y_autofocus -->
				<input
					bind:value={search}
					autofocus
					placeholder="Buscar respuesta…"
					aria-label="Buscar respuesta guardada"
					class="h-8 w-full rounded-md border bg-background px-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
					onkeydown={(e) => {
						if (e.key === 'Escape') open = false;
						if (e.key === 'Enter' && filtered[0]) {
							e.preventDefault();
							onpick(filtered[0]);
							open = false;
						}
					}}
				/>
			</div>
			{#if macros === null}
				<p class="p-4 text-sm text-muted-foreground">Cargando…</p>
			{:else if filtered.length === 0}
				<p class="p-4 text-sm text-muted-foreground">
					{macros.length === 0 ? 'Aún no hay respuestas guardadas.' : 'Nada coincide.'}
					<a href={resolve('/(app)/macros')} class="text-primary underline dark:text-honey"
						>Gestionarlas</a
					>
				</p>
			{:else}
				<ul class="max-h-64 overflow-y-auto py-1">
					{#each filtered as macro (macro.id)}
						<li>
							<button
								type="button"
								class="block w-full px-3 py-2 text-left text-sm hover:bg-muted"
								onclick={() => {
									onpick(macro);
									open = false;
								}}
							>
								<span class="block font-medium">{macro.title}</span>
								<span class="line-clamp-1 text-xs text-muted-foreground">{macro.body}</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	{/if}
</div>
