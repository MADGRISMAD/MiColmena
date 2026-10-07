<script lang="ts">
	import { resolve } from '$app/paths';
	import XIcon from '@lucide/svelte/icons/x';
	import PortalLink from '#lib/components/portal-link.svelte';
	import type { OrgUsage } from '#lib/api/index.js';

	let { org }: { org: OrgUsage } = $props();

	// Se puede cerrar; se recuerda por empresa en este navegador.
	const key = $derived(`micolmena_bienvenida_${org.id}`);
	let hidden = $state(false);
	$effect(() => {
		try {
			hidden = localStorage.getItem(key) === 'oculta';
		} catch {
			hidden = false;
		}
	});

	function dismiss() {
		hidden = true;
		try {
			localStorage.setItem(key, 'oculta');
		} catch {
			// Sin almacenamiento: se vuelve a mostrar en la próxima visita.
		}
	}

	const steps = [
		{
			title: 'Revisa tus categorías y plazos de atención',
			text: 'Ya dejamos unas categorías y SLA de ejemplo; ajústalos a tu negocio.',
			href: resolve('/(app)/admin/settings'),
			action: 'Configurar'
		},
		{
			title: 'Escribe tu primer artículo de ayuda',
			text: 'Se sugiere a tus clientes antes de que abran un ticket.',
			href: resolve('/help/new'),
			action: 'Escribir'
		},
		{
			title: 'Crea una respuesta guardada',
			text: 'Para las preguntas que te hacen todos los días.',
			href: resolve('/(app)/macros'),
			action: 'Crear'
		},
		{
			title: 'Suma a tu equipo',
			text: 'Da de alta agentes o amplía tu plan cuando lo necesites.',
			href: resolve('/(app)/admin/users'),
			action: 'Usuarios'
		}
	];
</script>

{#if !hidden}
	<section
		class="bg-honeycomb relative mb-6 rounded-2xl border border-honey/50 bg-card p-6"
		aria-labelledby="welcome-title"
	>
		<button
			type="button"
			class="absolute top-4 right-4 rounded-md p-1.5 text-muted-foreground hover:bg-muted"
			onclick={dismiss}
			aria-label="Ocultar la bienvenida"
		>
			<XIcon class="size-4" aria-hidden="true" />
		</button>
		<h2 id="welcome-title" class="text-lg font-semibold">¡Bienvenido a MiColmena, {org.name}!</h2>
		<p class="mt-1 text-sm text-muted-foreground">
			Comparte tu portal y tus clientes ya pueden abrir tickets. Estos pasos te ayudan a dejarlo
			listo.
		</p>
		<div class="mt-5 max-w-2xl">
			<p class="mb-2 text-sm font-medium">1. Tu portal de soporte</p>
			<PortalLink slug={org.slug} />
		</div>
		<ol class="mt-5 grid gap-3 md:grid-cols-2">
			{#each steps as step, i (step.title)}
				<li class="flex items-start gap-3 rounded-xl border bg-background/70 p-4">
					<span
						class="hex grid size-7 shrink-0 place-items-center bg-honey text-xs font-bold text-honey-foreground"
						>{i + 2}</span
					>
					<span class="min-w-0 flex-1">
						<span class="block text-sm font-medium">{step.title}</span>
						<span class="block text-xs text-muted-foreground">{step.text}</span>
					</span>
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- href ya viene de resolve() -->
					<a
						href={step.href}
						class="shrink-0 text-sm font-medium text-primary hover:underline dark:text-honey"
						>{step.action}</a
					>
				</li>
			{/each}
		</ol>
	</section>
{/if}
