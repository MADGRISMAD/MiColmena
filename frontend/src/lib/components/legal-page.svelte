<script lang="ts">
	import type { Snippet } from 'svelte';
	import { resolve } from '$app/paths';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import Logo from '#lib/components/logo.svelte';

	let { title, updated, children }: { title: string; updated: string; children: Snippet } =
		$props();
</script>

<svelte:head><title>{title} · MiColmena</title></svelte:head>

<div class="flex min-h-screen flex-col">
	<header class="border-b">
		<div class="mx-auto flex h-16 max-w-3xl items-center justify-between px-4">
			<a href={resolve('/')} aria-label="MiColmena, inicio"><Logo /></a>
			<nav class="flex gap-4 text-sm text-muted-foreground" aria-label="Documentos legales">
				<a href={resolve('/terminos')} class="hover:text-foreground">Términos</a>
				<a href={resolve('/privacidad')} class="hover:text-foreground">Privacidad</a>
			</nav>
		</div>
	</header>
	<main class="mx-auto w-full max-w-3xl flex-1 px-4 py-10">
		<div
			class="mb-8 flex gap-3 rounded-xl border border-amber-400 bg-amber-50 p-4 text-sm text-amber-950 dark:bg-amber-500/10 dark:text-amber-100"
			role="note"
		>
			<TriangleAlertIcon class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
			<p>
				<strong>Borrador.</strong> Este documento es una plantilla que debe revisar un abogado antes de
				usarse. Los datos entre [corchetes] los completa el responsable de MiColmena.
			</p>
		</div>
		<h1 class="text-3xl font-bold tracking-tight">{title}</h1>
		<p class="mt-2 text-sm text-muted-foreground">Última actualización: {updated}</p>
		<div class="legal mt-8">
			{@render children()}
		</div>
	</main>
</div>

<style>
	.legal :global(h2) {
		margin-top: 2rem;
		font-size: 1.15rem;
		font-weight: 600;
	}
	.legal :global(p),
	.legal :global(ul) {
		margin-top: 0.75rem;
		line-height: 1.7;
		color: var(--muted-foreground);
	}
	.legal :global(ul) {
		list-style: disc;
		padding-left: 1.25rem;
	}
	.legal :global(strong) {
		color: var(--foreground);
	}
</style>
