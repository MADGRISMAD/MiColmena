<script lang="ts">
	import type { Snippet } from 'svelte';
	import { resolve } from '$app/paths';
	import CheckIcon from '@lucide/svelte/icons/check';
	import Logo from '#lib/components/logo.svelte';

	let {
		title,
		description,
		children,
		footer,
		headline = 'El soporte de tu equipo, organizado como una colmena.',
		points = [
			'Cada solicitud en su celda: nada se pierde en el correo.',
			'Notas internas para coordinar al equipo sin que el cliente las vea.',
			'Prioridades y estados claros para atender primero lo urgente.'
		],
		wide = false
	}: {
		title: string;
		description: string;
		children: Snippet;
		footer: Snippet;
		/** Frase del panel de marca; en el portal de una empresa, su nombre. */
		headline?: string;
		points?: string[];
		/** Formulario más ancho (registro de empresa). */
		wide?: boolean;
	} = $props();
</script>

<div class="grid min-h-screen lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
	<!-- Panel de marca (solo en pantallas grandes) -->
	<aside
		class="bg-honeycomb-dark relative hidden flex-col justify-between overflow-hidden bg-sidebar p-10 text-sidebar-foreground lg:flex"
	>
		<div
			class="pointer-events-none absolute -top-40 -right-40 size-[28rem] rounded-full bg-honey/20 blur-3xl"
			aria-hidden="true"
		></div>
		<a href={resolve('/')} class="relative w-fit"><Logo tone="dark" /></a>

		<div class="relative max-w-md">
			<p class="text-3xl leading-tight font-semibold text-balance text-white">
				{headline}
			</p>
			<ul class="mt-8 grid gap-4">
				{#each points as point (point)}
					<li class="flex gap-3">
						<span
							class="hex mt-0.5 grid size-6 shrink-0 place-items-center bg-honey text-honey-foreground"
						>
							<CheckIcon class="size-3.5" aria-hidden="true" />
						</span>
						<span>{point}</span>
					</li>
				{/each}
			</ul>
		</div>

		<p class="relative text-sm text-sidebar-foreground/60">
			© {new Date().getFullYear()} MiColmena
		</p>
	</aside>

	<!-- Formulario -->
	<main class="flex items-center justify-center px-4 py-10 sm:px-6">
		<div class={wide ? 'w-full max-w-lg' : 'w-full max-w-sm'}>
			<a href={resolve('/')} class="mb-10 inline-flex lg:hidden"><Logo /></a>
			<h1 class="text-2xl font-bold tracking-tight">{title}</h1>
			<p class="mt-1 text-muted-foreground">{description}</p>
			<div class="mt-8">
				{@render children()}
			</div>
			<p class="mt-8 text-center text-sm text-muted-foreground">
				{@render footer()}
			</p>
		</div>
	</main>
</div>
